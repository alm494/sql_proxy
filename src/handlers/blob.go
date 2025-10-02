package handlers

import (
	"net/http"
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

	// TO DO

}
