package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/doug-martin/goqu/v9"
	"github.com/gregdel/pushover"
	"math"
	"time"
)

// Check will validate functionality of the tool by checking the deadman feed.
// This job will notify using pushover if the deadman feed is not working.
type Check struct {
	ScheduleOverride string

	DB            *sql.DB
	PushoverToken string
	PushoverApp   string
}

func (d *Check) Name() string {
	return "check"
}

func (d *Check) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	goquDB := goqu.New("postgres", d.DB)

	go func() {
		var err error
		defer func() {
			if err != nil {
				app := pushover.New(d.PushoverApp)
				recipient := pushover.NewRecipient(d.PushoverToken)
				message := pushover.NewMessage(err.Error())

				_, sendErr := app.SendMessage(message, recipient)
				if sendErr != nil {
					errCh <- sendErr
					return
				}

				errCh <- err
			}
		}()

		var item struct {
			ID        int64     `db:"id"`
			Title     string    `db:"title"`
			Body      string    `db:"body"`
			URL       string    `db:"url"`
			CreatedAt time.Time `db:"created_at"`
		}

		found, err := goquDB.From("webhookrss.items").
			Where(goqu.C("feed").Eq("deadman")).
			Order(goqu.I("created_at").Desc()).
			Limit(1).
			ScanStruct(&item)
		if err != nil {
			err = fmt.Errorf("failed to query deadman feed: %w", err)
			return
		}
		if !found {
			err = fmt.Errorf("deadman feed is empty")
			return
		}

		utc := time.Now().UTC()

		diff := utc.Sub(item.CreatedAt)

		if math.Abs(diff.Hours()) > 24 {
			err = fmt.Errorf("deadman feed is stale: %s", diff)
			return
		}

		doneCh <- true
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-errCh:
		return fmt.Errorf("job failed with error: %s", e)
	case <-doneCh:
		return nil
	}
}

func (d *Check) Timeout() time.Duration {
	return 15 * time.Second
}

func (d *Check) Schedule() string {
	if d.ScheduleOverride != "" {
		return d.ScheduleOverride
	}
	return "0 0 0 * * *"
}
