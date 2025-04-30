package main

import (
	"fmt"
	"net/http"

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
	app.Log.SetLevel(logrus.Level(app.GetEnvInt("LOG_LEVEL", 2)))
	bindAddress := app.GetEnvString("BIND_ADDR", "localhost")
	bindPort := app.GetEnvInt("BIND_PORT", 8080)
	db.MaxRows = uint32(app.GetEnvInt("MAX_ROWS", 10000))
	tlsCert := app.GetEnvString("TLS_CERT", "")
	tlsKey := app.GetEnvString("TLS_KEY", "")

	// Init connections handler map
	db.Handler.Init()

	// Scheduled maintenance task
	go db.Handler.RunMaintenance()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/connection", handlers.CreateConnection).Methods("POST")
	router.HandleFunc("/api/v1/connection", handlers.CloseConnection).Methods("DELETE")
	router.HandleFunc("/api/v1/query", handlers.SelectQuery).Methods("POST")
	router.HandleFunc("/api/v1/query", handlers.ExecuteQuery).Methods("PUT")
	router.HandleFunc("/api/v1/prepared", handlers.PrepareStatement).Methods("POST")
	router.HandleFunc("/api/v1/prepared/query", handlers.PreparedSelect).Methods("POST")
	router.HandleFunc("/api/v1/prepared/query", handlers.PreparedExecute).Methods("PUT")
	router.HandleFunc("/api/v1/prepared", handlers.ClosePreparedStatement).Methods("DELETE")
	router.HandleFunc("/healthz", handlers.Healthz).Methods("GET")
	router.HandleFunc("/readyz", handlers.Readyz).Methods("GET")
	router.HandleFunc("/livez", handlers.Livez).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	app.Log.Info("(c) 2025 Almaz Sharipov, MIT license, https://github.com/alm494/sql_proxy")
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
