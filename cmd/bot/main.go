// Command bot is the entrypoint of the booking Telegram bot.
//
// On startup it: loads configuration, applies pending database migrations,
// connects to PostgreSQL and Redis, wires up the user service and Telegram
// handlers, then runs the bot's long-polling loop until SIGINT/SIGTERM.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	botapp "github.com/happyinside-dev/booking-bot/internal/bot"
	"github.com/happyinside-dev/booking-bot/internal/bot/fsm"
	"github.com/happyinside-dev/booking-bot/internal/bot/handlers"
	"github.com/happyinside-dev/booking-bot/internal/config"
	"github.com/happyinside-dev/booking-bot/internal/database"
	"github.com/happyinside-dev/booking-bot/internal/logger"
	"github.com/happyinside-dev/booking-bot/internal/repository"
	"github.com/happyinside-dev/booking-bot/internal/service"
	"github.com/happyinside-dev/booking-bot/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal startup error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.AppEnv, cfg.LogLevel)
	slog.SetDefault(log)

	log.Info("starting booking-bot",
		slog.String("env", cfg.AppEnv),
		slog.Int("default_slot_interval_min", cfg.DefaultSlotIntervalMin),
	)

	// Root context cancelled on SIGINT/SIGTERM: this is the seed of the
	// graceful shutdown wired up fully in a later stage.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		return err
	}
	log.Info("migrations applied")

	pgPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pgPool.Close()
	log.Info("connected to postgres")

	redisClient, err := database.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := redisClient.Close(); cerr != nil {
			log.Error("error closing redis client", slog.Any("error", cerr))
		}
	}()
	log.Info("connected to redis")

	userRepo := repository.NewUserRepository(pgPool)
	userService := service.NewUserService(userRepo, log)

	businessRepo := repository.NewBusinessRepository(pgPool)
	memberRepo := repository.NewBusinessMemberRepository(pgPool)

	businessService := service.NewBusinessService(
		businessRepo,
		memberRepo,
		log,
		service.BusinessDefaults{
			SlotIntervalMin:     cfg.DefaultSlotIntervalMin,
			MinBookingNoticeMin: cfg.DefaultMinBookingNotice,
			MaxBookingDays:      cfg.DefaultMaxBookingDays,
		},
	)
	serviceRepo := repository.NewServiceRepository(pgPool)
	employeeRepo := repository.NewEmployeeRepository(pgPool)
	scheduleRepo := repository.NewScheduleRepository(pgPool)
	catalogService := service.NewCatalogService(serviceRepo, businessService)
	employeeService := service.NewEmployeeService(employeeRepo, businessService)
	scheduleService := service.NewScheduleService(businessRepo, serviceRepo, employeeRepo, scheduleRepo, businessService)
	bookingRepo := repository.NewBookingRepository(pgPool)
	bookingService := service.NewBookingService(bookingRepo, scheduleService, serviceRepo, employeeRepo, userRepo, businessRepo)

	type configuredBot struct {
		id, businessID  int64
		username, token string
	}
	var configured []configuredBot
	if len(cfg.BotTokens) > 0 {
		items, err := repository.NewBusinessBotRepository(pgPool).ListActive(ctx)
		if err != nil {
			return err
		}
		for _, item := range items {
			token, ok := cfg.BotTokens[item.TokenKey]
			if !ok || token == "" {
				return fmt.Errorf("missing token for active business bot %q", item.TokenKey)
			}
			configured = append(configured, configuredBot{id: item.ID, businessID: item.BusinessID, username: item.Username, token: token})
		}
		if len(configured) == 0 {
			return fmt.Errorf("no active business bots configured")
		}
	} else {
		businessID := cfg.BotBusinessID
		if businessID == 0 {
			businessID, err = businessRepo.SingleBusinessID(ctx)
			if err != nil {
				return err
			}
		}
		configured = append(configured, configuredBot{id: businessID, businessID: businessID, token: cfg.BotToken})
	}

	bots := make([]*botapp.Bot, 0, len(configured))
	senders := make(worker.StaticSenderRegistry, len(configured))
	for _, item := range configured {
		stateStore := fsm.NewRedisStore(redisClient, 30*time.Minute, strconv.FormatInt(item.id, 10))
		h := handlers.New(
			handlers.BotContext{BotID: item.id, BusinessID: item.businessID, Username: item.username},
			userService, businessService, catalogService, employeeService, scheduleService, bookingService, stateStore, log,
		)
		tgBot, err := botapp.New(item.token, h, log.With(slog.Int64("bot_id", item.id), slog.Int64("business_id", item.businessID)))
		if err != nil {
			return err
		}
		bots = append(bots, tgBot)
		senders[item.businessID] = tgBot
	}
	notificationWorker := worker.NewNotificationWorker(
		repository.NewNotificationRepository(pgPool), service.NewNotificationService(), senders,
		worker.NotificationConfig{PollInterval: cfg.NotificationPollInterval, BatchSize: cfg.NotificationBatchSize, Lease: cfg.NotificationLease, MaxAttempts: cfg.NotificationMaxAttempts, RetryDelay: cfg.NotificationRetryDelay}, log,
	)

	var wg sync.WaitGroup
	for _, tgBot := range bots {
		wg.Add(1)
		go func() { defer wg.Done(); tgBot.Start(ctx) }()
	}
	wg.Add(1)
	go func() { defer wg.Done(); notificationWorker.Run(ctx) }()
	<-ctx.Done()
	wg.Wait()

	log.Info("shutdown complete")

	return nil
}
