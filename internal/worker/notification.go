package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/model"
)

type NotificationRepository interface {
	ClaimDue(context.Context, time.Time, int, time.Duration) ([]model.Notification, error)
	IsDeliverable(context.Context, int64) (bool, error)
	MarkSent(context.Context, int64, time.Time) error
	MarkFailed(context.Context, int64, time.Time, int, time.Duration, string) error
}
type Formatter interface {
	Format(model.Notification) (string, error)
}
type MessageSender interface {
	SendText(context.Context, int64, string) error
}
type SenderRegistry interface {
	SenderForBusiness(int64) (MessageSender, bool)
}

type NotificationConfig struct {
	PollInterval time.Duration
	BatchSize    int
	Lease        time.Duration
	MaxAttempts  int
	RetryDelay   time.Duration
}
type NotificationWorker struct {
	repo      NotificationRepository
	formatter Formatter
	senders   SenderRegistry
	cfg       NotificationConfig
	log       *slog.Logger
	now       func() time.Time
}

func NewNotificationWorker(repo NotificationRepository, formatter Formatter, senders SenderRegistry, cfg NotificationConfig, log *slog.Logger) *NotificationWorker {
	return &NotificationWorker{repo: repo, formatter: formatter, senders: senders, cfg: cfg, log: log, now: time.Now}
}

func (w *NotificationWorker) Run(ctx context.Context) {
	w.log.Info("notification worker started")
	defer w.log.Info("notification worker stopped")
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.processBatch(ctx); err != nil && ctx.Err() == nil {
			w.log.Error("notification batch failed", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w *NotificationWorker) processBatch(ctx context.Context) error {
	now := w.now()
	items, err := w.repo.ClaimDue(ctx, now, w.cfg.BatchSize, w.cfg.Lease)
	if err != nil {
		return err
	}
	for _, item := range items {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.processOne(ctx, item)
	}
	return nil
}
func (w *NotificationWorker) processOne(ctx context.Context, n model.Notification) {
	deliverable, err := w.repo.IsDeliverable(ctx, n.ID)
	if err != nil {
		w.fail(ctx, n, err)
		return
	}
	if !deliverable {
		return
	}
	sender, ok := w.senders.SenderForBusiness(n.BusinessID)
	if !ok {
		w.fail(ctx, n, fmt.Errorf("no active bot for business %d", n.BusinessID))
		return
	}
	message, err := w.formatter.Format(n)
	if err != nil {
		w.fail(ctx, n, err)
		return
	}
	if err := sender.SendText(ctx, n.TelegramID, message); err != nil {
		w.fail(ctx, n, err)
		return
	}
	if err := w.repo.MarkSent(ctx, n.ID, w.now()); err != nil {
		w.log.Error("failed to mark notification sent", slog.Int64("notification_id", n.ID), slog.Any("error", err))
	}
}
func (w *NotificationWorker) fail(ctx context.Context, n model.Notification, err error) {
	message := err.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	if markErr := w.repo.MarkFailed(ctx, n.ID, w.now(), w.cfg.MaxAttempts, w.cfg.RetryDelay, message); markErr != nil {
		w.log.Error("failed to record notification error", slog.Int64("notification_id", n.ID), slog.Any("error", markErr))
		return
	}
	w.log.Warn("notification delivery failed", slog.Int64("notification_id", n.ID), slog.Int("attempt", n.Attempts), slog.Any("error", err))
}

type StaticSenderRegistry map[int64]MessageSender

func (r StaticSenderRegistry) SenderForBusiness(id int64) (MessageSender, bool) {
	sender, ok := r[id]
	return sender, ok
}
