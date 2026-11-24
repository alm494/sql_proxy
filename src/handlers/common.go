package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sql-proxy/src/app"
	"sql-proxy/src/db"
)

// Shape of a normal response. Clients depend on this — don't break it lightly
type ResponseEnvelope struct {
	ApiVersion     string           `json:"api_version"`
	ConnectionId   string           `json:"connection_id"`
	Info           string           `json:"info"`
	RowsCount      uint32           `json:"rows_count"`
	ExceedsMaxRows bool             `json:"exceeds_max_rows"`
	Rows           []map[string]any `json:"rows"`
}

// Same idea, but for the moments things go wrong (they do)
type ErrorEnvelope struct {
	ApiVersion string `json:"api_version"`
	Error      string `json:"error"`
	Status     int    `json:"status"`
}

// Gatekeeper for every endpoint: the API-Version header must match or you get 501 and a polite goodbye
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

// Central place to log an error and ship the standard JSON envelope out. One format for all our failures
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

// Take a result set and pack it into the envelope our clients expect
func tableResponce(w http.ResponseWriter, connId string, rows *sql.Rows) {

	columns, err := rows.Columns()
	if err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tableData, rowsCount, exceedsMaxRows := convertRows(rows, &columns)

	var envelope ResponseEnvelope
	envelope.ApiVersion = app.ApiVersion
	envelope.ConnectionId = connId
	envelope.RowsCount = rowsCount
	envelope.ExceedsMaxRows = exceedsMaxRows
	envelope.Rows = *tableData

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(envelope)

}

// Walk the result set row by row and cut it at MaxRows — that's why big queries end up with exceeds flag set. We mean well
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
		if rowsCount >= db.MaxRows {
			exceedsMaxRows = true
			break
		}
		tableData = append(tableData, entry)
		rowsCount++
	}
	return &tableData, rowsCount, exceedsMaxRows

}
