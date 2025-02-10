package main

import (
	"fmt"
	"net/http"
	"os"

	"sql-proxy/src/db"
	"sql-proxy/src/handlers"
	"sql-proxy/src/utils"
	"sql-proxy/src/version"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	var err error

	// Application params taken from OS environment
	utils.Log.SetLevel(logrus.Level(utils.GetIntEnvOrDefault("LOG_LEVEL", 2)))
	bindAddress := os.Getenv("BIND_ADDR")
	bindPort := utils.GetIntEnvOrDefault("BIND_PORT", 8080)
	db.MaxRows = utils.GetIntEnvOrDefault("MAX_ROWS", 10000)
	tlsCert := os.Getenv("TLS_CERT")
	tlsKey := os.Getenv("TLS_KEY")

	// Scheduled maintenance task
	go db.DbHandler.RunMaintenance()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/query/select", handlers.GetQuery).Methods("GET")
	router.HandleFunc("/api/v1/query/execute", handlers.ExecuteQuery).Methods("POST")
	router.HandleFunc("/api/v1/connection/create", handlers.CreateConnection).Methods("POST")
	router.HandleFunc("/api/v1/connection/delete", handlers.CloseConnection).Methods("DELETE")
	router.HandleFunc("/healthz", handlers.Healthz).Methods("GET")
	router.HandleFunc("/readyz", handlers.Readyz).Methods("GET")
	router.HandleFunc("/livez", handlers.Livez).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	utils.Log.WithFields(logrus.Fields{
		"build_version": version.BuildVersion,
		"build_time":    version.BuildTime,
	}).Info("Starting server sql-proxy:")

	utils.Log.WithFields(logrus.Fields{
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
		utils.Log.WithError(err).Fatal("Fatal error occurred, service stopped")
	}
}
