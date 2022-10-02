package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Clean will remove items overflow items from feeds with more than 50 items
type Clean struct {
	ScheduleOverride string

	DB *sql.DB
}

func (c *Clean) Name() string {
	return "clean"
}

func (c *Clean) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	go func() {
		_, err := c.DB.Exec(`
delete from webhookrss.items where id in (
  select id from webhookrss.items where feed in (
    select feed from (select feed, count(id) as count from webhookrss.items group by feed) as counts
    where count > 50
  )
  order by created_at desc
  offset 50
) `)
		if err != nil {
			errCh <- fmt.Errorf("failed to clean old items: %w", err)
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

func (c *Clean) Timeout() time.Duration {
	return 15 * time.Second
}

func (c *Clean) Schedule() string {
	if c.ScheduleOverride != "" {
		return c.ScheduleOverride
	}
	return "0 0 0 * * *"
}
