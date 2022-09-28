package tool

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/charlieegan3/tool-webhook-rss/pkg/tool/handlers"
	"github.com/charlieegan3/toolbelt/pkg/apis"
	"github.com/gorilla/mux"
)

//go:embed migrations
var webhookRSSToolMigrations embed.FS

// WebhookRSS is an example tool which demonstrates the use of the database feature
type WebhookRSS struct {
	db *sql.DB
}

func (d *WebhookRSS) Name() string {
	return "webhook-rss"
}

func (d *WebhookRSS) FeatureSet() apis.FeatureSet {
	return apis.FeatureSet{
		HTTP:     true,
		Database: true,
	}
}

func (d *WebhookRSS) HTTPPath() string {
	return "webhook-rss"
}

// SetConfig is a no-op for this tool
func (d *WebhookRSS) SetConfig(config map[string]any) error {
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
