package handlers

import (
	"io"
	"net/http"

	"sql-proxy/src/app"
	"sql-proxy/src/db"
)

// Run an ad-hoc SELECT and send the result set back in our envelope format — simple as that
func SelectQuery(w http.ResponseWriter, r *http.Request) {

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

	rows, err := dbConn.Query(sqlQuery)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tableResponce(w, connId, rows)

}

// Same thing but for statements that don't return anything — empty body on success is a feature
func ExecuteQuery(w http.ResponseWriter, r *http.Request) {

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

	_, err := dbConn.Exec(sqlQuery)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
	}

}

// Read the Connection-Id header plus raw SQL from the body — both must show up or we stop right there
func parseQueryHttpHeadersAndBody(w http.ResponseWriter, r *http.Request) (string, string, bool) {

	connId := r.Header.Get("Connection-Id")

	body, err := io.ReadAll(r.Body)
	if err != nil || connId == "" || len(body) == 0 {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return "", "", false
	}
	defer r.Body.Close()

	sqlQuery := string(body)
	app.Logger.Debugf("SQL query received: sql=%s, connection_id=%s", sqlQuery, connId)

	return connId, sqlQuery, true

}
