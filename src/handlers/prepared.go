package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"sql-proxy/src/app"
	"sql-proxy/src/db"
)

func PrepareStatement(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	connId, sqlQuery, ok := parsePrepareStatementHttpHeadersAndBody(w, r)
	if !ok {
		return
	}

	conn, ok := db.Handler.GetById(connId, true)
	if !ok {
		errorResponce(w, "Invalid connection id", http.StatusForbidden)
		return
	}

	stmt, err := conn.Prepare(sqlQuery)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmtId, ok := db.Handler.PutPreparedStatement(connId, stmt)
	if !ok {
		errorResponce(w, "Error saving statement into pool", http.StatusInternalServerError)
		return
	}

	if _, err = w.Write([]byte(stmtId)); err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
	}

}

func PreparedSelect(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	connId, stmtId, params, ok := parseExecuteStatementHttpHeadersAndBody(w, r)
	if !ok {
		return
	}

	dbStmt, ok := db.Handler.GetPreparedStatement(connId, stmtId)
	if !ok {
		errorResponce(w, "Prepared statement not found", http.StatusForbidden)
		return
	}
	rows, err := dbStmt.Query(params...)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tableResponce(w, rows)

}

func PreparedExecute(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	connId, stmtId, params, ok := parseExecuteStatementHttpHeadersAndBody(w, r)
	if !ok {
		return
	}

	dbStmt, ok := db.Handler.GetPreparedStatement(connId, stmtId)
	if !ok {
		errorResponce(w, "Prepared statement not found", http.StatusForbidden)
	}
	_, err := dbStmt.Exec(params...)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
	}

}

func ClosePreparedStatement(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	connId := r.Header.Get("Connection-Id")
	stmtId := r.Header.Get("Statement-Id")

	if connId == "" || stmtId == "" {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return
	}

	app.Logger.Debugf("Delete prepared statememt received: connection_id=%s, prepared_statement=%s", connId, stmtId)

	if ok := db.Handler.ClosePreparedStatement(connId, stmtId); !ok {
		errorResponce(w, "Forbidden", http.StatusForbidden)
	}

}

func parsePrepareStatementHttpHeadersAndBody(w http.ResponseWriter, r *http.Request) (string, string, bool) {

	connId := r.Header.Get("Connection-Id")

	body, err := io.ReadAll(r.Body)
	if err != nil || connId == "" || len(body) == 0 {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return "", "", false
	}
	defer r.Body.Close()

	sqlQuery := string(body)
	app.Logger.Debugf("Prepared statement received: sql=%s, connection_id=%s", sqlQuery, connId)

	return connId, sqlQuery, true

}

func parseExecuteStatementHttpHeadersAndBody(w http.ResponseWriter, r *http.Request) (string, string, []any, bool) {

	connId := r.Header.Get("Connection-Id")
	stmtId := r.Header.Get("Statement-Id")

	body, err := io.ReadAll(r.Body)
	if err != nil || connId == "" || stmtId == "" {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return "", "", nil, false
	}
	defer r.Body.Close()

	var params []any
	if len(body) == 0 || string(body) == "null" {
		params = nil
	} else {
		err = json.Unmarshal(body, &params)
		if err != nil {
			errorResponce(w, "Bad request", http.StatusBadRequest)
			return "", "", nil, false
		}
	}

	app.Logger.Debugf("Execute prepared statement received: connection_id=%s, statement_id=%s", connId, stmtId)

	return connId, stmtId, params, true

}
