package tool

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/Jeffail/gabs/v2"
	"github.com/charlieegan3/toolbelt/pkg/apis"
	"github.com/gorilla/mux"

	"github.com/charlieegan3/tool-webhook-rss/pkg/tool/handlers"
	"github.com/charlieegan3/tool-webhook-rss/pkg/tool/jobs"
)

//go:embed migrations
var webhookRSSToolMigrations embed.FS

// WebhookRSS is a tool to create RSS feeds from webhooks, it has a handler to accept new items and display feeds.
// There are also a number of jobs to keep the database clean and check that the tool is still working.
type WebhookRSS struct {
	config *gabs.Container
	db     *sql.DB
}

func (d *WebhookRSS) Name() string {
	return "webhook-rss"
}

func (d *WebhookRSS) FeatureSet() apis.FeatureSet {
	return apis.FeatureSet{
		Config:   true,
		HTTP:     true,
		Database: true,
		Jobs:     true,
	}
}

func (d *WebhookRSS) HTTPPath() string {
	return "webhook-rss"
}

func (d *WebhookRSS) SetConfig(config map[string]any) error {
	d.config = gabs.Wrap(config)

	return nil
}

func (d *WebhookRSS) DatabaseMigrations() (*embed.FS, string, error) {
	return &webhookRSSToolMigrations, "migrations", nil
}

func (d *WebhookRSS) DatabaseSet(db *sql.DB) {
	d.db = db
}

func (d *WebhookRSS) HTTPAttach(router *mux.Router) error {
	if d.db == nil {
		return fmt.Errorf("database not set")
	}

	// handler for the creation of new items in feeds
	router.HandleFunc(
		"/feeds/{feed}/items",
		handlers.BuildItemCreateHandler(d.db),
	).Methods("POST")

	// handler used to serve rss clients
	router.HandleFunc(
		"/feeds/{feed}.rss",
		handlers.BuildFeedGetHandler(d.db),
	).Methods("GET")

	return nil
}

func (d *WebhookRSS) Jobs() ([]apis.Job, error) {
	var j []apis.Job
	var path string
	var ok bool

	// load deadman config
	path = "jobs.deadman.endpoint"
	deadmanEndpoint, ok := d.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "jobs.deadman.schedule"
	deadmanSchedule, ok := d.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	// load deadman check config
	path = "jobs.deadman-check.schedule"
	deadmanCheckSchedule, ok := d.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "jobs.deadman-check.pushover_token"
	deadmanCheckPushoverToken, ok := d.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "jobs.deadman-check.pushover_app"
	deadmanCheckPushoverApp, ok := d.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	// load clean config
	path = "jobs.clean.schedule"
	cleanSchedule, ok := d.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	// load clean check config
	path = "jobs.clean-check.schedule"
	cleanCheckSchedule, ok := d.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "jobs.clean-check.endpoint"
	cleanCheckEndpoint, ok := d.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	return []apis.Job{
		&jobs.DeadMan{
			Endpoint:         deadmanEndpoint,
			ScheduleOverride: deadmanSchedule,
		},
		&jobs.DeadmanCheck{
			DB:               d.db,
			ScheduleOverride: deadmanCheckSchedule,
			PushoverApp:      deadmanCheckPushoverApp,
			PushoverToken:    deadmanCheckPushoverToken,
		},
		&jobs.Clean{
			DB:               d.db,
			ScheduleOverride: cleanSchedule,
		},
		&jobs.CleanCheck{
			DB:               d.db,
			ScheduleOverride: cleanCheckSchedule,
			Endpoint:         cleanCheckEndpoint,
		},
	}, nil
}
func (d *WebhookRSS) ExternalJobsFuncSet(f func(job apis.ExternalJob) error) {
}
