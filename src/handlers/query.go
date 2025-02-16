package handlers

import (
	"database/sql"
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

// Converts SQL query result to json
func convertRows(rows *sql.Rows, columns *[]string) (*[]map[string]interface{}, uint32, bool) {
	var rowsCount uint32 = 0
	colsCount := len(*columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, colsCount)
	valuePtrs := make([]interface{}, colsCount)
	exceedsMaxRows := false

	for rows.Next() {
		for i := range *columns {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range *columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		if rowsCount > db.MaxRows {
			exceedsMaxRows = true
			break
		}
		tableData = append(tableData, entry)
		rowsCount++
	}
	return &tableData, rowsCount, exceedsMaxRows
}
