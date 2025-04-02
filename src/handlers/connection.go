package handlers

import (
	"encoding/json"
	"net/http"
	"sql-proxy/src/db"
)

func CreateConnection(w http.ResponseWriter, r *http.Request) {

	var dbConnInfo db.DbConnInfo

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&dbConnInfo); err != nil {
		errorResponce(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	if connGuid, ok := db.Handler.GetByParams(&dbConnInfo); !ok {
		errorResponce(w, "Failed to get SQL connection", http.StatusInternalServerError)
	} else if _, err := w.Write([]byte(connGuid)); err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
	}

}

func CloseConnection(w http.ResponseWriter, r *http.Request) {

	if ok := checkApiVersion(w, r); !ok {
		return
	}

	connId := r.Header.Get("Connection-Id")
	if connId == "" {
		errorResponce(w, "Bad request", http.StatusBadRequest)
		return
	}
	db.Handler.Delete(connId)

}
