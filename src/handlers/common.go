package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sql-proxy/src/app"
	"sql-proxy/src/db"
)

type ResponseEnvelope struct {
	ApiVersion     string           `json:"api_version"`
	ConnectionId   string           `json:"connection_id"`
	Info           string           `json:"info"`
	RowsCount      uint32           `json:"rows_count"`
	ExceedsMaxRows bool             `json:"exceeds_max_rows"`
	Rows           []map[string]any `json:"rows"`
}

type ErrorEnvelope struct {
	ApiVersion string `json:"api_version"`
	Error      string `json:"error"`
	Status     int    `json:"status"`
}

func checkApiVersion(w http.ResponseWriter, r *http.Request) bool {

	apiVersion := r.Header.Get("API-Version")
	if apiVersion != app.ApiVersion {
		message := "Unsupported API version"
		app.Logger.Error(message)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(ErrorEnvelope{
			ApiVersion: app.ApiVersion,
			Error:      message,
			Status:     http.StatusNotImplemented,
		})
		return false
	} else {
		return true
	}

}

func errorResponce(w http.ResponseWriter, message string, httpStatus int) {

	app.Logger.Error(message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	errResp := ErrorEnvelope{
		ApiVersion: app.ApiVersion,
		Error:      http.StatusText(httpStatus),
		Status:     httpStatus,
	}
	json.NewEncoder(w).Encode(errResp)

}

func tableResponce(w http.ResponseWriter, rows *sql.Rows) {

	columns, err := rows.Columns()
	if err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tableData, rowsCount, exceedsMaxRows := convertRows(rows, &columns)

	var envelope ResponseEnvelope
	envelope.ApiVersion = app.ApiVersion
	envelope.RowsCount = rowsCount
	envelope.ExceedsMaxRows = exceedsMaxRows
	envelope.Rows = *tableData

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(envelope)

}

// Converts SQL query result to JSON array
func convertRows(rows *sql.Rows, columns *[]string) (*[]map[string]any, uint32, bool) {

	var rowsCount uint32 = 0
	colsCount := len(*columns)
	tableData := make([]map[string]any, 0)
	values := make([]any, colsCount)
	valuePtrs := make([]any, colsCount)
	exceedsMaxRows := false

	for rows.Next() {
		for i := range *columns {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]any)
		for i, col := range *columns {
			var v any
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
