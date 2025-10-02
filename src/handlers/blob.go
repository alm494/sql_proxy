package handlers

import (
	"io"
	"net/http"
	"sql-proxy/src/app"
	"sql-proxy/src/db"
)

func ReadBlob(w http.ResponseWriter, r *http.Request) {

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

	var data []byte
	err := dbConn.QueryRow(sqlQuery).Scan(&data)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)

}

func WriteBlob(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	maxSize := int64(32 << 20) // 32 MB, change here if required
	connId, sqlQuery, data, ok := parseQueryHttpHeadersAndMultipartBody(r, maxSize)
	if !ok {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return
	}

	dbConn, ok := db.Handler.GetById(connId, true)
	if !ok {
		errorResponce(w, "Invalid connection id", http.StatusForbidden)
		return
	}

	_, err := dbConn.Exec(sqlQuery, data)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
	}

}

func parseQueryHttpHeadersAndMultipartBody(r *http.Request, maxSize int64) (string, string, []byte, bool) {

	connId := r.Header.Get("Connection-Id")

	err := r.ParseMultipartForm(maxSize)
	if err != nil {
		return "", "", nil, false
	}

	sqlQuery := r.FormValue("sql_query")
	if sqlQuery == "" {
		return "", "", nil, false
	}

	file, _, err := r.FormFile("binary_data")
	if err != nil {
		return "", "", nil, false
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", nil, false
	}

	app.Logger.Debugf("SQL query received: sql=%s, connection_id=%s", sqlQuery, connId)

	return connId, sqlQuery, data, true

}
