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

		var items []toolAPIs.PayloadNewItem
		err := json.NewDecoder(request.Body).Decode(&items)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		var records []goqu.Record

		for _, item := range items {
			if item.Title == "" {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}

			if len(item.Title) > 500 {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}

			if len(item.Body) > 100000 {
				writer.WriteHeader(http.StatusBadRequest)
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

		_, err = ins.Executor().Exec()
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		writer.WriteHeader(http.StatusOK)
	}
}
