package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"

	"sql-proxy/src/app"
	"sql-proxy/src/db"

	"github.com/sirupsen/logrus"
)

func SelectQuery(w http.ResponseWriter, r *http.Request) {

	connId := r.Header.Get("Connection-Id")
	query, err := url.QueryUnescape(r.Header.Get("SQL-Statement"))

	if err != nil || connId == "" || query == "" {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"sql":           query,
		"connection_id": connId,
	}).Debug("SQL query received:")

	// Search existings connection in the pool
	dbConn, ok := db.Handler.GetById(connId, true)
	if !ok {
		errorResponce(w, "Failed to get SQL connection", http.StatusForbidden)
		return
	}

	rows, err := dbConn.Query(query)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tableData, rowsCount, exceedsMaxRows := convertRows(rows, &columns)

	var envelope ResponseEnvelope
	envelope.RowsCount = rowsCount
	envelope.ExceedsMaxRows = exceedsMaxRows
	envelope.Rows = *tableData

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(envelope)
}

func ExecuteQuery(w http.ResponseWriter, r *http.Request) {

	connId := r.Header.Get("Connection-Id")
	query, err := url.QueryUnescape(r.Header.Get("SQL-Statement"))

	if err != nil || connId == "" || query == "" {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"sql":           query,
		"connection_id": connId,
	}).Debug("SQL query received:")

	dbConn, ok := db.Handler.GetById(connId, true)
	if !ok {
		errorResponce(w, "Invalid connection id", http.StatusForbidden)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"sql":           query,
		"connection_id": connId,
	}).Debug("SQL execute query received:")

	_, err = dbConn.Exec(query)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
	}
}
