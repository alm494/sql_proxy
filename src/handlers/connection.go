package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"sql-proxy/src/db"
	"sql-proxy/src/utils"
)

func CreateConnection(w http.ResponseWriter, r *http.Request) {
	var dbConnInfo db.DbConnInfo

	err := json.NewDecoder(r.Body).Decode(&dbConnInfo)
	if err != nil {
		errorMsg := "Error decoding JSON"
		utils.Log.Error(errorMsg)
		http.Error(w, errorMsg, http.StatusBadRequest)
		return
	}

	connGuid, ok := db.DbHandler.GetByParams(&dbConnInfo)

	if !ok {
		errorMsg := "Failed to get SQL connection"
		utils.Log.Error(errorMsg)
		http.Error(w, errorMsg, http.StatusInternalServerError)
	} else {
		_, err := w.Write([]byte(connGuid))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func CloseConnection(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	db.DbHandler.Delete(string(bodyBytes))
}
