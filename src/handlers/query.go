package handlers

import (
	"encoding/json"
	"net/http"

	"sql-proxy/src/app"
	"sql-proxy/src/db"

	"github.com/sirupsen/logrus"
)

func SelectQuery(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("sql")
	conn := r.URL.Query().Get("connection_id")
	if query == "" || conn == "" {
		errorText := "Missing parameter"
		app.Log.Error(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"sql":           query,
		"connection_id": conn,
	}).Debug("SQL query received:")

	// Search existings connection in the pool
	dbConn, ok := db.Handler.GetById(conn, true)
	if !ok {
		errorText := "Failed to get SQL connection"
		app.Log.Error(errorText, ": ", conn)
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

	var requestBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	conn, ok := db.Handler.GetById(requestBody["connection_id"].(string), true)
	if !ok {
		http.Error(w, "Invalid connection id", http.StatusBadRequest)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"sql":           requestBody["sql"].(string),
		"connection_id": requestBody["connection_id"].(string),
	}).Debug("SQL execute query received:")

	_, err = conn.Exec(requestBody["sql"].(string))
	if err != nil {
		http.Error(w, "Invalid SQL query", http.StatusBadRequest)
	}
}
