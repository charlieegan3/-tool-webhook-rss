package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DeadMan will send events into a given feed at regular intervals.
// Another job will test if this is functional and alert if not
type DeadMan struct {
	Endpoint         string
	ScheduleOverride string
}

func (d *DeadMan) Name() string {
	return "deadman-feed"
}

func (d *DeadMan) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	go func() {
		datab := []map[string]string{
			{
				"title": "Dead Man Pulse",
				"body":  "",
				"url":   "",
			},
		}

		b, err := json.Marshal(datab)
		if err != nil {
			errCh <- fmt.Errorf("failed to form dead man item JSON: %s", err)
			return
		}

		client := &http.Client{}
		req, err := http.NewRequest("POST", d.Endpoint, bytes.NewBuffer(b))
		if err != nil {
			errCh <- fmt.Errorf("failed to build request for dead man item: %s", err)
			return
		}

		req.Header.Add("Content-Type", "application/json; charset=utf-8")

		resp, err := client.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("failed to send request for dead man item: %s", err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			errCh <- fmt.Errorf("failed to send request: non 200OK response")
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

func (d *DeadMan) Timeout() time.Duration {
	return 15 * time.Second
}

func (d *DeadMan) Schedule() string {
	if d.ScheduleOverride != "" {
		return d.ScheduleOverride
	}
	return "0 0 0 * * *"
}
