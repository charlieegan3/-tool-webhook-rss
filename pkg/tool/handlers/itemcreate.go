package handlers

import (
	"database/sql"
	"encoding/json"
	toolAPIs "github.com/charlieegan3/tool-webhook-rss/pkg/apis"
	"github.com/doug-martin/goqu/v9"
	"github.com/gorilla/mux"
	"net/http"
)

func BuildItemCreateHandler(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	goquDB := goqu.New("postgres", db)

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		feed, ok := vars["feed"]
		if !ok || feed == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("feed var missing"))
			return
		}

		if !feedRegex.MatchString(feed) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("feed didn't match regex"))
			return
		}

		var items []toolAPIs.PayloadNewItem
		arrErr := json.NewDecoder(r.Body).Decode(&items)
		if arrErr != nil {
			// here we handle the case where a single item is sent.
			// regrettably, the apple shortcuts app can't send arrays, so we have to handle single items here.
			var item toolAPIs.PayloadNewItem
			err := json.NewDecoder(r.Body).Decode(&item)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("failed to parse JSON data as as item array or item object"))
			}
			items = []toolAPIs.PayloadNewItem{item}
		}

		var records []goqu.Record

		for _, item := range items {
			if item.Title == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("title can't be blank"))
				return
			}

			if len(item.Title) > 500 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("title too long"))
				return
			}

			if len(item.Body) > 100000 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("body too long"))
				return
			}

			records = append(records, goqu.Record{
				"feed":  feed,
				"title": item.Title,
				"body":  item.Body,
				"url":   item.URL,
			})
		}

		ins := goquDB.Insert("webhookrss.items").Rows(records)

		_, err := ins.Executor().Exec()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
