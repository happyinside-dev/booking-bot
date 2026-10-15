package repository_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/database"
	"github.com/happyinside-dev/booking-bot/internal/model"
	"github.com/happyinside-dev/booking-bot/internal/repository"
)

// TestPostgresIntegration exercises the guarantees that mocks cannot prove:
// embedded migrations, exclusion constraints, advisory-lock serialization,
// transactional notification creation/cancellation and SKIP LOCKED claiming.
// It is opt-in so the regular unit-test suite stays hermetic.
func TestPostgresIntegration(t *testing.T) {
	adminDSN := os.Getenv("TEST_DATABASE_URL")
	if adminDSN == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := newMigratedTestDatabase(t, ctx, adminDSN)

	var version int64
	if err := pool.QueryRow(ctx, `SELECT max(version_id) FROM goose_db_version WHERE is_applied`).Scan(&version); err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != 13 {
		t.Fatalf("migration version = %d, want 13", version)
	}

	var businessID, userID, serviceID, employeeID, employee2ID int64
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO businesses (name,description,address,timezone) VALUES ('Test','Description','Address','Europe/Moscow') RETURNING id`), &businessID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO users (telegram_id,first_name) VALUES (900000001,'Client') RETURNING id`), &userID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO services (business_id,name,duration_min,price) VALUES ($1,'Service',60,1500.50) RETURNING id`, businessID), &serviceID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO employees (business_id,name) VALUES ($1,'First') RETURNING id`, businessID), &employeeID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO employees (business_id,name) VALUES ($1,'Second') RETURNING id`, businessID), &employee2ID)

	bookings := repository.NewBookingRepository(pool)
	schedules := repository.NewScheduleRepository(pool)
	notifications := repository.NewNotificationRepository(pool)
	start := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Minute)

	t.Run("business and owner are created atomically", func(t *testing.T) {
		businesses := repository.NewBusinessRepository(pool)
		_, err := businesses.CreateWithOwner(ctx, &model.Business{Name: "Rollback", Timezone: "UTC", SlotIntervalMin: 30, MinBookingNoticeMin: 60, MaxBookingDays: 30}, 999999999)
		if err == nil {
			t.Fatal("expected missing owner foreign-key error")
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM businesses WHERE name='Rollback'`).Scan(&count); err != nil {
			t.Fatalf("count rolled-back businesses: %v", err)
		}
		if count != 0 {
			t.Fatalf("orphan businesses = %d, want 0", count)
		}
	})

	t.Run("two concurrent bookings cannot overlap", func(t *testing.T) {
		params := repository.CreateBookingParams{BusinessID: businessID, EmployeeID: employeeID, ServiceID: serviceID, UserID: userID, StartAt: start}
		errs := runTogether(func() error { _, err := bookings.Create(ctx, params); return err }, func() error { _, err := bookings.Create(ctx, params); return err })
		assertOneSuccessOneConflict(t, errs)
	})

	t.Run("booking and blocked slot are serialized", func(t *testing.T) {
		params := repository.CreateBookingParams{BusinessID: businessID, EmployeeID: employee2ID, ServiceID: serviceID, UserID: userID, StartAt: start}
		slot := &model.BlockedSlot{EmployeeID: employee2ID, StartAt: start, EndAt: start.Add(time.Hour), Reason: "test"}
		errs := runTogether(func() error { _, err := bookings.Create(ctx, params); return err }, func() error { _, err := schedules.CreateBlockedSlot(ctx, slot); return err })
		assertOneSuccessOneConflict(t, errs)
	})

	t.Run("notifications are claimed once and cancelled transactionally", func(t *testing.T) {
		var bookingID int64
		if err := pool.QueryRow(ctx, `SELECT id FROM bookings WHERE employee_id=$1`, employeeID).Scan(&bookingID); err != nil {
			t.Fatalf("find booking: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE notifications SET scheduled_at=$1 WHERE booking_id=$2`, time.Now().Add(-time.Minute), bookingID); err != nil {
			t.Fatalf("make notifications due: %v", err)
		}

		var batches [2][]model.Notification
		errs := runTogether(
			func() error {
				var err error
				batches[0], err = notifications.ClaimDue(ctx, time.Now(), 1, time.Minute)
				return err
			},
			func() error {
				var err error
				batches[1], err = notifications.ClaimDue(ctx, time.Now(), 1, time.Minute)
				return err
			},
		)
		for _, err := range errs {
			if err != nil {
				t.Fatalf("claim notification: %v", err)
			}
		}
		if len(batches[0]) != 1 || len(batches[1]) != 1 || batches[0][0].ID == batches[1][0].ID {
			t.Fatalf("claims must contain two distinct notifications: %#v %#v", batches[0], batches[1])
		}
		if err := bookings.Cancel(ctx, bookingID, userID, time.Now()); err != nil {
			t.Fatalf("cancel booking: %v", err)
		}
		var cancelled int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE booking_id=$1 AND status='CANCELLED'`, bookingID).Scan(&cancelled); err != nil {
			t.Fatalf("count cancelled notifications: %v", err)
		}
		if cancelled != 2 {
			t.Fatalf("cancelled notifications = %d, want 2", cancelled)
		}
	})
}

func newMigratedTestDatabase(t *testing.T, ctx context.Context, adminDSN string) *pgxpool.Pool {
	t.Helper()
	parsed, err := url.Parse(adminDSN)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	name := fmt.Sprintf("booking_bot_test_%d", time.Now().UnixNano())
	admin, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close(ctx)
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = admin.Exec(cleanupCtx, `DROP DATABASE `+pgx.Identifier{name}.Sanitize()+` WITH (FORCE)`)
		_ = admin.Close(cleanupCtx)
	})

	parsed.Path = "/" + name
	testDSN := parsed.String()
	if err := database.RunMigrations(testDSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func mustScan(t *testing.T, row pgx.Row, destinations ...any) {
	t.Helper()
	if err := row.Scan(destinations...); err != nil {
		t.Fatalf("seed database: %v", err)
	}
}

func runTogether(first, second func() error) [2]error {
	start := make(chan struct{})
	var wg sync.WaitGroup
	var result [2]error
	for i, fn := range []func() error{first, second} {
		wg.Add(1)
		go func(index int, call func() error) { defer wg.Done(); <-start; result[index] = call() }(i, fn)
	}
	close(start)
	wg.Wait()
	return result
}

func assertOneSuccessOneConflict(t *testing.T, errs [2]error) {
	t.Helper()
	successes, conflicts := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, apperr.ErrBookingConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}
}
