package handlers

import (
	"net/http"
	"net/url"
	"sql-proxy/src/app"
	"sql-proxy/src/db"

	"github.com/sirupsen/logrus"
)

func PrepareStatement(w http.ResponseWriter, r *http.Request) {

	connId := r.Header.Get("Connection-Id")
	preparedStatement, err := url.QueryUnescape(r.Header.Get("Prepared-Statement"))

	if err != nil || connId == "" || preparedStatement == "" {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"prepared_statement": preparedStatement,
		"connection_id":      connId,
	}).Debug("Prepare statement received:")

	conn, ok := db.Handler.GetById(connId, true)
	if !ok {
		errorResponce(w, "Invalid connection id", http.StatusForbidden)
		return
	}

	stmt, err := conn.Prepare(preparedStatement)
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
	}

	stmtId, ok := db.Handler.PutPreparedStatement(connId, stmt)
	if !ok {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
	}

	if _, err = w.Write([]byte(stmtId)); err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
	}

}

func PreparedSelect(w http.ResponseWriter, r *http.Request) {

	// to do

}

func PreparedExecute(w http.ResponseWriter, r *http.Request) {

	// to do
}

func ClosePreparedStatement(w http.ResponseWriter, r *http.Request) {

	connId := r.Header.Get("Connection-Id")
	stmtId := r.Header.Get("Statement-Id")

	if connId == "" || stmtId == "" {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return
	}

	app.Log.WithFields(logrus.Fields{
		"connection_id":      connId,
		"prepared_statement": stmtId,
	}).Debug("Delete prepared statememt received:")

	if ok := db.Handler.ClosePreparedStatement(connId, stmtId); !ok {
		errorResponce(w, "Forbidden", http.StatusForbidden)
	}

}
