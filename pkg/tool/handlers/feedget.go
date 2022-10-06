package handlers

import (
	"database/sql"
	"fmt"
	"github.com/doug-martin/goqu/v9"
	"github.com/gorilla/feeds"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
	"time"
)

func BuildFeedGetHandler(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	goquDB := goqu.New("postgres", db)

	return func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)

		feed, ok := vars["feed"]
		if !ok || feed == "" {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		if feedRegex.MatchString(feed) == false {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		responseFeed := &feeds.Feed{
			Title:       feed,
			Link:        &feeds.Link{Href: request.URL.String()},
			Description: fmt.Sprintf("webhook-rss feed %q", feed),
			Created:     time.Now(),
		}

		var items []struct {
			ID        int64     `db:"id"`
			Title     string    `db:"title"`
			Body      string    `db:"body"`
			URL       string    `db:"url"`
			CreatedAt time.Time `db:"created_at"`
		}

		err := goquDB.From("webhookrss.items").
			Where(goqu.C("feed").Eq(feed), goqu.C("created_at").Lt("NOW()")).
			Order(goqu.I("created_at").Desc()).
			Limit(50).
			ScanStructs(&items)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(items) > 0 {
			responseFeed.Created = items[0].CreatedAt
		}

		for _, item := range items {
			responseFeed.Items = append(responseFeed.Items,
				&feeds.Item{
					Id:          fmt.Sprintf("%s/items/%d", strings.TrimSuffix(request.URL.String(), ".rss"), item.ID),
					Title:       item.Title,
					Link:        &feeds.Link{Href: item.URL},
					Description: item.Body,
					Created:     item.CreatedAt,
				})
		}

		atom, err := responseFeed.ToAtom()
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		writer.Write([]byte(atom))
	}
}
