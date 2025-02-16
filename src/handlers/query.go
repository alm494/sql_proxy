package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"sql-proxy/src/db"
	"sql-proxy/src/utils"

	"github.com/sirupsen/logrus"
)

func GetQuery(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	conn := r.URL.Query().Get("conn")
	if query == "" || conn == "" {
		errorText := "Missing parameter"
		utils.Log.Error(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	utils.Log.WithFields(logrus.Fields{
		"query": query,
		"conn":  conn,
	}).Debug("SQL query received:")

	// Search existings connection in the pool
	dbConn, ok := db.DbHandler.GetById(conn, true)
	if !ok {
		errorText := "Failed to get SQL connection"
		utils.Log.Error(errorText, ": ", conn)
		http.Error(w, errorText, http.StatusForbidden)
		return
	}

	rows, err := dbConn.Query(query)
	if err != nil {
		utils.Log.WithError(err).Error("SQL query error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		utils.Log.WithError(err).Error("Invalid query return value")
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
	var payload ExecuteQueryEnvelope
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	conn, ok := db.DbHandler.GetById(payload.Conn, true)
	if !ok {
		http.Error(w, "Invalid connection id", http.StatusBadRequest)
		return
	}
	_, err = conn.Exec(payload.SQL)
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
