package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/doug-martin/goqu/v9"
)

type FeedCheck struct {
	DB *sql.DB

	Endpoint         string
	ScheduleOverride string

	Feeds []interface{}
}

func (c *FeedCheck) Name() string {
	return "feed-check"
}

func (c *FeedCheck) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	goquDB := goqu.New("postgres", c.DB)

	go func() {
		fmt.Println("here")
		var rows []struct {
			Feed string    `db:"feed"`
			Age  time.Time `db:"created_at"`
		}

		sel := goquDB.From("webhookrss.items").Select("feed", goqu.MAX("created_at").As("created_at")).GroupBy("feed")
		err := sel.Executor().ScanStructs(&rows)
		if err != nil {
			errCh <- fmt.Errorf("failed to get feed ages: %w", err)
			return
		}

		for _, row := range rows {
			for _, feed := range c.Feeds {
				feedData, ok := feed.(map[string]interface{})
				if !ok {
					errCh <- fmt.Errorf("failed to parse feed data")
					return
				}

				feedName, ok := feedData["name"].(string)
				if !ok {
					errCh <- fmt.Errorf("failed to parse feed name")
					return
				}
				if feedName == row.Feed {
					maxAgeString, ok := feedData["max_age"].(string)
					if !ok {
						errCh <- fmt.Errorf("failed to parse max_age for feed %s", feedName)
						return
					}

					maxAge, err := time.ParseDuration(maxAgeString)
					if err != nil {
						errCh <- fmt.Errorf("failed to parse max_age for feed %s: %w", feedName, err)
						return
					}

					if time.Now().UTC().Sub(row.Age) > maxAge {
						log.Println("Alerting for feed", feedName)
						err := alert(
							c.Endpoint,
							"Feed Stale Error",
							fmt.Sprintf("Feed %s has not been updated in over %s", feedName, maxAge),
						)
						if err != nil {
							errCh <- fmt.Errorf("failed to send alert for feed %s: %w", feedName, err)
							return
						}
					}
				}
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

func (c *FeedCheck) Timeout() time.Duration {
	return 15 * time.Second
}

func (c *FeedCheck) Schedule() string {
	if c.ScheduleOverride != "" {
		return c.ScheduleOverride
	}
	return "0 0 0 * * *"
}

func alert(webhookRSSEndpoint, title, message string) error {
	datab := []map[string]string{
		{
			"title": title,
			"body":  message,
			"url":   "",
		},
	}

	b, err := json.Marshal(datab)
	if err != nil {
		return fmt.Errorf("failed to form alert item JSON: %s", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", webhookRSSEndpoint, bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("failed to build request for alert item: %s", err)
	}

	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request for alert item: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send request: non 200OK response: %d", resp.StatusCode)
	}

	return nil
}
