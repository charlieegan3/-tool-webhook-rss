package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/charlieegan3/toolbelt/pkg/database"
	"github.com/charlieegan3/toolbelt/pkg/tool"
	"github.com/spf13/viper"

	webhookTool "github.com/charlieegan3/tool-webhook-rss/pkg/tool"
)

// this is an example use of a Tool Belt showing the registration of an example Hello World tool
func main() {
	viper.SetConfigFile(os.Args[1])
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
		os.Exit(1)
	}

	params := viper.GetStringMapString("database.params")
	connectionString := viper.GetString("database.connectionString")
	db, err := database.Init(connectionString, params, params["dbname"], false)
	if err != nil {
		log.Fatalf("failed to init DB: %s", err)
	}

	tb := tool.NewBelt()
	tb.SetDatabase(db)

	// this might be loaded from disk in some real example
	toolCfg := viper.Get("tools").(map[string]interface{})
	tb.SetConfig(toolCfg)

	err = tb.AddTool(&webhookTool.WebhookRSS{})
	if err != nil {
		log.Fatalf("failed to add tool: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-c:
			cancel()
		}
	}()

	tb.RunJobs(ctx)

	// tb.RunServer(ctx, "0.0.0.0", "3000")
}
