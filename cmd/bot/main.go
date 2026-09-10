// Command bot is the entrypoint of the booking Telegram bot.
//
// At this stage (Infrastructure) it only wires up configuration, logging,
// and connections to PostgreSQL and Redis, then waits for a shutdown
// signal. Telegram integration is added in Stage 2.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourname/booking-bot/internal/config"
	"github.com/yourname/booking-bot/internal/database"
	"github.com/yourname/booking-bot/internal/logger"
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

	log.Info("booking-bot is up, waiting for shutdown signal")

	<-ctx.Done()

	log.Info("shutdown signal received, stopping gracefully")

	return nil
}
