package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sql-proxy/src/app"
	"sql-proxy/src/db"
	"sql-proxy/src/handlers"

	"github.com/gorilla/mux"
	"github.com/kardianos/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var svcLogger service.Logger

type program struct {
	exit chan struct{}
}

func (p *program) Start(s service.Service) error {
	app.Logger.Info("Starting sql-proxy service...")
	p.exit = make(chan struct{})
	go p.run()
	return nil
}

func (p *program) run() {

	// Application params taken from OS environment
	bindAddress := app.GetEnvString("BIND_ADDR", "localhost")
	if bindAddress == "*" {
		bindAddress = ""
	}
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

	app.Logger.Info("(c) 2025 Almaz Sharipov, MIT license, https://github.com/alm494/sql_proxy  ")
	app.Logger.Infof("build_version=%s, build_time=%s", app.BuildVersion, app.BuildTime)
	app.Logger.Infof("Server started with the following parameters: "+
		"bind_port=%d, bind_address=%s, tls_cert=%s, tls_key=%s", bindPort, bindAddress, tlsCert, tlsKey)

	addr := fmt.Sprintf("%s:%d", bindAddress, bindPort)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		var err error
		if len(tlsCert) > 0 && len(tlsKey) > 0 {
			err = srv.ListenAndServeTLS(tlsCert, tlsKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			app.Logger.Errorf("Fatal error occurred, service stopped: %v", err)
		}
	}()

	// Wait for exit signal
	<-p.exit

	// Shutdown server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.Logger.Errorf("Server shutdown failed: %v", err)
	} else {
		app.Logger.Info("Server exited properly")
	}
}

func (p *program) Stop(s service.Service) error {
	app.Logger.Info("Stopping sql-proxy service...")
	close(p.exit)
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "sql-proxy",
		DisplayName: "SQL Proxy Service",
		Description: "A lightweight REST service designed to replace ADODB calls in legacy software systems that support web requests",
		Arguments:   []string{},
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	svcLogger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		// Handle service commands: install, start, stop, uninstall
		err := service.Control(s, os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Run as a regular app
	if !service.Interactive() {
		app.InitLogger(app.NewServiceLogger(svcLogger))
		err = s.Run()
		if err != nil {
			svcLogger.Error(err)
		}
	} else {
		// Run in console mode
		app.InitLogger(app.NewConsoleLogger())
		fmt.Println("Running in console mode...")
		prg.Start(nil)

		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("Received interrupt signal, shutting down...")
		prg.Stop(nil)
	}
}
