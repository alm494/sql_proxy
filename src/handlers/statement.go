package handlers

import (
	"encoding/json"
	"net/http"
)

func PrepareStatement(w http.ResponseWriter, r *http.Request) {
	var payload ExecuteQueryEnvelope
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	/*
	   conn, ok := app.DbHandler.GetById(payload.Conn, true)

	   	if !ok {
	   		http.Error(w, "Invalid connection id", http.StatusBadRequest)
	   		return
	   	}

	   stmt, err := conn.Prepare(payload.SQL)

	   	if err != nil {
	   		http.Error(w, "Failed to prepare statement", http.StatusBadRequest)
	   	}
	*/
}

func SelectStatement(w http.ResponseWriter, r *http.Request) {
	// to do
}

func ExecuteStatement(w http.ResponseWriter, r *http.Request) {
	// to do
}
