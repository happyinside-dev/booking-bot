// Command bot is the entrypoint of the booking Telegram bot.
//
// On startup it: loads configuration, applies pending database migrations,
// connects to PostgreSQL and Redis, wires up the user service and Telegram
// handlers, then runs the bot's long-polling loop until SIGINT/SIGTERM.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	botapp "github.com/yourname/booking-bot/internal/bot"
	"github.com/yourname/booking-bot/internal/bot/handlers"
	"github.com/yourname/booking-bot/internal/config"
	"github.com/yourname/booking-bot/internal/database"
	"github.com/yourname/booking-bot/internal/logger"
	"github.com/yourname/booking-bot/internal/repository"
	"github.com/yourname/booking-bot/internal/service"
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
	h := handlers.New(userService, log)

	tgBot, err := botapp.New(cfg.BotToken, h, log)
	if err != nil {
		return err
	}

	// Blocks (long polling) until ctx is cancelled by SIGINT/SIGTERM, then
	// returns so the deferred pool/redis cleanup above runs.
	tgBot.Start(ctx)

	log.Info("shutdown complete")

	return nil
}
