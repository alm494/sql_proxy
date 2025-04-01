package handlers

import (
	"encoding/json"
	"net/http"
	"sql-proxy/src/db"
)

func PrepareStatement(w http.ResponseWriter, r *http.Request) {
	var requestBody map[string]any

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		errorResponce(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	conn, ok := db.Handler.GetById(requestBody["connection_id"].(string), true)
	if !ok {
		errorResponce(w, "Invalid connection id", http.StatusBadRequest)
		return
	}

	stmt, err := conn.Prepare(requestBody["sql"].(string))
	if err != nil {
		errorResponce(w, err.Error(), http.StatusBadRequest)
	}

	stmt_id, ok := db.Handler.PutPreparedStatement(requestBody["connection_id"].(string), stmt)
	if !ok {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
	}

	if _, err = w.Write([]byte(stmt_id)); err != nil {
		errorResponce(w, err.Error(), http.StatusInternalServerError)
	}
}

func PreparedSelect(w http.ResponseWriter, r *http.Request) {
	var requestBody map[string]any

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		errorResponce(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// to do
}

func PreparedExecute(w http.ResponseWriter, r *http.Request) {
	// to do
}

func ClosePreparedStatement(w http.ResponseWriter, r *http.Request) {
	// to do
}
