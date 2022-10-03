package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/doug-martin/goqu/v9"
	"net/http"
	"strings"
	"time"
)

type CleanCheck struct {
	DB *sql.DB

	Endpoint         string
	ScheduleOverride string
}

func (c *CleanCheck) Name() string {
	return "clean-check"
}

func (c *CleanCheck) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	goquDB := goqu.New("postgres", c.DB)

	go func() {
		var rows []struct {
			Feed  string `db:"feed"`
			Count int    `db:"count"`
		}

		counts := goquDB.From("webhookrss.items").Select("feed", goqu.COUNT("id").As("count")).GroupBy("feed")

		sel := goquDB.From(counts).Select("feed", "count").Where(goqu.C("count").Gt(75)).Order(goqu.C("count").Desc())

		err := sel.Executor().ScanStructs(&rows)
		if err != nil {
			errCh <- fmt.Errorf("failed to get state to check if clean: %w", err)
			return
		}

		if len(rows) > 0 {
			items := []string{}
			for _, row := range rows {
				items = append(items, fmt.Sprintf("<li>%s has %d items</li>", row.Feed, row.Count))
			}
			body := fmt.Sprintf("<ul>%s</ul>", strings.Join(items, "\n"))

			datab := []map[string]string{
				{
					"title": "Clean Check Failed",
					"body":  body,
					"url":   "",
				},
			}

			b, err := json.Marshal(datab)
			if err != nil {
				errCh <- fmt.Errorf("failed to form item JSON: %s", err)
				return
			}

			client := &http.Client{}
			req, err := http.NewRequest("POST", c.Endpoint, bytes.NewBuffer(b))
			if err != nil {
				errCh <- fmt.Errorf("failed to build request for clean warning: %s", err)
				return
			}

			req.Header.Add("Content-Type", "application/json; charset=utf-8")

			resp, err := client.Do(req)
			if err != nil {
				errCh <- fmt.Errorf("failed to send request with clean warning: %s", err)
				return
			}

			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("failed to send request with clean warning: non 200OK response")
				return
			}
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

func (c *CleanCheck) Timeout() time.Duration {
	return 15 * time.Second
}

func (c *CleanCheck) Schedule() string {
	if c.ScheduleOverride != "" {
		return c.ScheduleOverride
	}
	return "0 0 0 * * *"
}
