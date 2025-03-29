package main

import (
	"fmt"
	"net/http"
	"os"

	"sql-proxy/src/app"
	"sql-proxy/src/db"
	"sql-proxy/src/handlers"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	var err error

	// Application params taken from OS environment
	app.Log.SetLevel(logrus.Level(app.GetIntEnvOrDefault("LOG_LEVEL", 2)))
	bindAddress := os.Getenv("BIND_ADDR")
	bindPort := app.GetIntEnvOrDefault("BIND_PORT", 8080)
	db.MaxRows = app.GetIntEnvOrDefault("MAX_ROWS", 10000)
	tlsCert := os.Getenv("TLS_CERT")
	tlsKey := os.Getenv("TLS_KEY")

	// Scheduled maintenance task
	go db.Handler.RunMaintenance()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/connection", handlers.CreateConnection).Methods("POST")
	router.HandleFunc("/api/v1/connection", handlers.CloseConnection).Methods("DELETE")
	router.HandleFunc("/api/v1/query", handlers.SelectQuery).Methods("GET")
	router.HandleFunc("/api/v1/query", handlers.ExecuteQuery).Methods("POST")
	router.HandleFunc("/api/v1/prepared", handlers.PrepareStatement).Methods("POST")
	router.HandleFunc("/api/v1/prepared/query", handlers.SelectStatement).Methods("GET")
	router.HandleFunc("/api/v1/prepared/query", handlers.ExecuteStatement).Methods("POST")
	router.HandleFunc("/api/v1/prepared", handlers.CloseStatement).Methods("DELETE")
	router.HandleFunc("/healthz", handlers.Healthz).Methods("GET")
	router.HandleFunc("/readyz", handlers.Readyz).Methods("GET")
	router.HandleFunc("/livez", handlers.Livez).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	app.Log.WithFields(logrus.Fields{
		"build_version": app.BuildVersion,
		"build_time":    app.BuildTime,
	}).Info("Starting server sql-proxy:")

	app.Log.WithFields(logrus.Fields{
		"bind_port":    bindPort,
		"bind_address": bindAddress,
		"tls_cert":     tlsCert,
		"tls_key":      tlsKey,
	}).Info("Server started with the following parameters:")

	addr := fmt.Sprintf("%s:%d", bindAddress, bindPort)
	if len(tlsCert) > 0 && len(tlsKey) > 0 {
		err = http.ListenAndServeTLS(addr, tlsCert, tlsKey, router)
	} else {
		err = http.ListenAndServe(addr, router)
	}
	if err != nil {
		app.Log.WithError(err).Fatal("Fatal error occurred, service stopped")
	}
}
