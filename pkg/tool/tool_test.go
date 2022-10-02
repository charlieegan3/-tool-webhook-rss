package tool

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	toolbeltAPIs "github.com/charlieegan3/toolbelt/pkg/apis"
	"github.com/charlieegan3/toolbelt/pkg/database/databasetest"
	"github.com/charlieegan3/toolbelt/pkg/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/charlieegan3/tool-webhook-rss/pkg/apis"
	"github.com/charlieegan3/tool-webhook-rss/pkg/tool/jobs"
)

func TestToolWebhookRSSSuite(t *testing.T) {
	s := &databasetest.DatabaseSuite{
		ConfigPath: "../../config.test.yaml",
	}

	s.Setup(t)

	s.AddDependentSuite(&ToolWebhookRSSSuite{DB: s.DB})

	s.Run(t)
}

type ToolWebhookRSSSuite struct {
	suite.Suite
	DB *sql.DB
}

func (s *ToolWebhookRSSSuite) Run(t *testing.T) {
	suite.Run(t, s)
}

func (s *ToolWebhookRSSSuite) TestJobsDeadMan() {
	t := s.T()

	tb := tool.NewBelt()

	tb.SetDatabase(s.DB)

	toolWebhookRSS := &WebhookRSS{
		loadedJobs: []toolbeltAPIs.Job{
			&jobs.DeadMan{
				ScheduleOverride: "* * * * * *",
				Endpoint:         "http://localhost:9032/webhook-rss/feeds/deadman/items",
			},
		},
	}

	err := tb.AddTool(toolWebhookRSS)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tb.RunServer(ctx, "0.0.0.0", "9032")
	go tb.RunJobs(ctx)

	// allow the jobs to run and create some entries
	time.Sleep(2 * time.Second)

	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost:9032",
			Path:   "/webhook-rss/feeds/deadman.rss",
		},
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Dead Man Pulse")
}

func (s *ToolWebhookRSSSuite) TestHTTP() {
	t := s.T()

	// create a new toolbelt to test the tool
	tb := tool.NewBelt()
	tb.SetDatabase(s.DB)
	webhookRSSTool := &WebhookRSS{}

	// register the tool with the toolbelt
	err := tb.AddTool(webhookRSSTool)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// start the toolbelt server to test the tool's http functions
	go func() {
		tb.RunServer(ctx, "0.0.0.0", "9032")
	}()

	// first, send some items to some feeds
	for _, feed := range []string{"feed1", "feed2"} {
		for _, item := range []string{"item1", "item2", "item3"} {
			payload := []apis.PayloadNewItem{
				{
					Title: item,
					Body:  fmt.Sprintf("body for item %s", item),
					URL:   "https://example.com",
				},
			}

			jsonData, err := json.Marshal(payload)
			require.NoError(t, err)

			req := &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "http",
					Host:   "localhost:9032",
					Path:   fmt.Sprintf("/webhook-rss/feeds/%s/items", feed),
				},
				Body: io.NopCloser(bytes.NewBuffer(jsonData)),
			}
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		}
	}

	// next, fetch some items from those same feeds
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost:9032",
			Path:   "/webhook-rss/feeds/feed1.rss",
		},
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "<id>/webhook-rss/feeds/feed1.rss</id>")
	assert.Contains(t, string(body), "<title>item1</title>")
	assert.Contains(t, string(body), "<id>/webhook-rss/feeds/feed1/items/")
	assert.Contains(t, string(body), `<link href="https://example.com" rel="alternate"></link>`)
	assert.Contains(t, string(body), `<summary type="html">body for item item1</summary>`)
	assert.Contains(t, string(body), "<title>item2</title>")
	assert.Contains(t, string(body), "<title>item3</title>")

	// check that the down migrations also work
	err = tb.DatabaseDownMigrate(webhookRSSTool)
	require.NoError(t, err)
}
