package handlers

import (
	"encoding/json"
	"net/http"

	"sql-proxy/src/app"
	"sql-proxy/src/db"

	"github.com/sirupsen/logrus"
)

func SelectQuery(w http.ResponseWriter, r *http.Request) {

	conn_id := r.Header.Get("Connection-Id")
	query := r.Header.Get("SQL-Statement")
	if conn_id == "" || query == "" {
		errorText := "Bad request"
		app.Log.Error(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"sql":           query,
		"connection_id": conn_id,
	}).Debug("SQL query received:")

	// Search existings connection in the pool
	dbConn, ok := db.Handler.GetById(conn_id, true)
	if !ok {
		errorText := "Failed to get SQL connection"
		app.Log.Error(errorText, ": ", conn_id)
		http.Error(w, errorText, http.StatusForbidden)
		return
	}

	rows, err := dbConn.Query(query)
	if err != nil {
		app.Log.WithError(err).Error("SQL query error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		app.Log.WithError(err).Error("Invalid query return value")
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	conn_id := r.Header.Get("Connection-Id")
	query := r.Header.Get("SQL-Statement")
	if conn_id == "" || query == "" {
		errorText := "Bad request"
		app.Log.Error(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	dbConn, ok := db.Handler.GetById(conn_id, true)
	if !ok {
		http.Error(w, "Invalid connection id", http.StatusForbidden)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"sql":           query,
		"connection_id": conn_id,
	}).Debug("SQL execute query received:")

	_, err := dbConn.Exec(query)
	if err != nil {
		http.Error(w, "Invalid SQL query", http.StatusBadRequest)
	}
}
