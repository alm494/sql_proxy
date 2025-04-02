package handlers

import (
	"io"
	"net/http"

	"sql-proxy/src/app"
	"sql-proxy/src/db"

	"github.com/sirupsen/logrus"
)

func SelectQuery(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	connId, sqlQuery, ok := parseQueryHttpHeadersAndBody(w, r)
	if !ok {
		return
	}

	dbConn, ok := db.Handler.GetById(connId, true)
	if !ok {
		errorResponce(w, "Invalid connection id", http.StatusForbidden)
		return
	}

	rows, err := dbConn.Query(sqlQuery)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tableResponce(w, rows)

}

func ExecuteQuery(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	connId, sqlQuery, ok := parseQueryHttpHeadersAndBody(w, r)
	if !ok {
		return
	}

	dbConn, ok := db.Handler.GetById(connId, true)
	if !ok {
		errorResponce(w, "Invalid connection id", http.StatusForbidden)
		return
	}

	_, err := dbConn.Exec(sqlQuery)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
	}

}

func parseQueryHttpHeadersAndBody(w http.ResponseWriter, r *http.Request) (string, string, bool) {

	connId := r.Header.Get("Connection-Id")

	body, err := io.ReadAll(r.Body)
	if err != nil || connId == "" || len(body) == 0 {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return "", "", false
	}
	defer r.Body.Close()

	sqlQuery := string(body)
	app.Log.WithFields(logrus.Fields{
		"sql":           sqlQuery,
		"connection_id": connId,
	}).Debug("SQL query received:")

	return connId, sqlQuery, true

}
