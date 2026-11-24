package handlers

import (
	"io"
	"net/http"
	"sql-proxy/src/app"
	"sql-proxy/src/db"
	"strings"
)

const maxBlobSize int64 = 32 << 20 // Hard ceiling at 32 MB — blobs come to rest in memory on the way, so we stay honest about it

// Pull a binary blob out of the database and spit it back as-is — no JSON ceremony here
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

	if int64(len(data)) > maxBlobSize {
		errorResponce(w, "Data too large", http.StatusRequestEntityTooLarge)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)

}

// Take the uploaded file and shove its bytes into the database via the query's parameter slot
func WriteBlob(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	connId, sqlQuery, data, ok := parseQueryHttpHeadersAndMultipartBody(r, maxBlobSize)
	if !ok {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return
	}

	dbConn, ok := db.Handler.GetById(connId, true)
	if !ok {
		errorResponce(w, "Invalid connection id", http.StatusForbidden)
		return
	}

	params := []any{data}
	if !strings.Contains(sqlQuery, "?") && !strings.Contains(sqlQuery, "$1") {
		sqlQuery = strings.TrimSpace(sqlQuery)
		if !strings.HasSuffix(sqlQuery, ";") {
			sqlQuery += " ?"
		} else {
			sqlQuery += "?"
		}
	}

	_, err := dbConn.Exec(sqlQuery, params...)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Pull apart the multipart form: sql_query field plus binary_data file. Missing anything means no deal
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
