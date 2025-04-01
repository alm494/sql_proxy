package handlers

import (
	"net/http"
	"sql-proxy/src/app"
)

func errorResponce(w http.ResponseWriter, message string, httpStatus int) {
	app.Log.Error(message)
	http.Error(w, message, httpStatus)
}
