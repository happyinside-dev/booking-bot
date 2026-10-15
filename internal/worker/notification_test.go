package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/model"
)

type notificationRepoFake struct {
	items                 []model.Notification
	deliverable           bool
	claimErr, errorToMark error
	sent, failed          []int64
	failureMessage        string
}

func (f *notificationRepoFake) ClaimDue(context.Context, time.Time, int, time.Duration) ([]model.Notification, error) {
	return f.items, f.claimErr
}
func (f *notificationRepoFake) IsDeliverable(context.Context, int64) (bool, error) {
	return f.deliverable, nil
}
func (f *notificationRepoFake) MarkSent(_ context.Context, id int64, _ time.Time) error {
	f.sent = append(f.sent, id)
	return f.errorToMark
}
func (f *notificationRepoFake) MarkFailed(_ context.Context, id int64, _ time.Time, _ int, _ time.Duration, message string) error {
	f.failed = append(f.failed, id)
	f.failureMessage = message
	return f.errorToMark
}

type formatterFake struct{ err error }

func (f formatterFake) Format(model.Notification) (string, error) { return "reminder", f.err }

type senderFake struct {
	err      error
	messages []string
}

func (f *senderFake) SendText(_ context.Context, _ int64, message string) error {
	f.messages = append(f.messages, message)
	return f.err
}

func newTestWorker(repo *notificationRepoFake, sender *senderFake) *NotificationWorker {
	w := NewNotificationWorker(repo, formatterFake{}, StaticSenderRegistry{7: sender}, NotificationConfig{PollInterval: time.Second, BatchSize: 10, Lease: time.Minute, MaxAttempts: 3, RetryDelay: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	w.now = func() time.Time { return time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC) }
	return w
}

func TestNotificationWorkerMarksSent(t *testing.T) {
	t.Parallel()
	repo := &notificationRepoFake{deliverable: true, items: []model.Notification{{ID: 1, BusinessID: 7, TelegramID: 42}}}
	sender := &senderFake{}
	if err := newTestWorker(repo, sender).processBatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.messages) != 1 || len(repo.sent) != 1 || len(repo.failed) != 0 {
		t.Fatalf("messages=%d sent=%v failed=%v", len(sender.messages), repo.sent, repo.failed)
	}
}
func TestNotificationWorkerRetriesFailure(t *testing.T) {
	t.Parallel()
	repo := &notificationRepoFake{deliverable: true, items: []model.Notification{{ID: 2, BusinessID: 7, TelegramID: 42}}}
	sender := &senderFake{err: errors.New("telegram unavailable")}
	if err := newTestWorker(repo, sender).processBatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.failed) != 1 || repo.failureMessage != "telegram unavailable" {
		t.Fatalf("failed=%v message=%q", repo.failed, repo.failureMessage)
	}
}
func TestNotificationWorkerSkipsCancelled(t *testing.T) {
	t.Parallel()
	repo := &notificationRepoFake{deliverable: false, items: []model.Notification{{ID: 3, BusinessID: 7}}}
	sender := &senderFake{}
	if err := newTestWorker(repo, sender).processBatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.messages) != 0 || len(repo.sent) != 0 || len(repo.failed) != 0 {
		t.Fatal("cancelled notification was processed")
	}
}
func TestNotificationWorkerReportsClaimError(t *testing.T) {
	t.Parallel()
	expected := errors.New("database down")
	repo := &notificationRepoFake{claimErr: expected}
	err := newTestWorker(repo, &senderFake{}).processBatch(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("error=%v", err)
	}
}
