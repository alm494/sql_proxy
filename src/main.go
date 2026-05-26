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

const shutdownTimeout = 15 * time.Second

type program struct {
	httpServer *http.Server
	exit       chan struct{}
}

func (p *program) Start(s service.Service) error {
	if !service.Interactive() {
		svcLogger, loggerErr := s.Logger(nil)
		if loggerErr != nil {
			return fmt.Errorf("failed to get service logger: %w", loggerErr)
		}
		app.InitLogger(app.NewServiceLogger(svcLogger))
	} else {
		app.InitLogger(app.NewConsoleLogger())
	}

	app.Logger.Info("Starting sql-proxy service...")

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	db.Handler.Init()
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
	router.HandleFunc("/api/v1/blob", handlers.ReadBlob).Methods("POST")
	router.HandleFunc("/api/v1/blob", handlers.WriteBlob).Methods("PUT")
	router.HandleFunc("/healthz", handlers.Healthz).Methods("GET")
	router.HandleFunc("/readyz", handlers.Readyz).Methods("GET")
	router.HandleFunc("/livez", handlers.Livez).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf("%s:%d", cfg.BindAddress, cfg.BindPort)
	p.httpServer = &http.Server{
		Addr:    addr,
		Handler: router,
	}
	p.exit = make(chan struct{})

	tlsEnabled := cfg.TLSCert != "" && cfg.TLSKey != ""
	app.Logger.Infof("(c) 2025-2026 Almaz Sharipov, MIT license, https://github.com/alm494/sql_proxy")
	app.Logger.Infof("build_version=%s, build_time=%s", app.BuildVersion, app.BuildTime)
	app.Logger.Infof("Server starting: bind=%s, max_conn=%d, max_rows=%d, tls=%v",
		addr, db.MaxConnections, db.MaxRows, tlsEnabled)

	go func() {
		var listenErr error
		if tlsEnabled {
			listenErr = p.httpServer.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			listenErr = p.httpServer.ListenAndServe()
		}

		if listenErr != nil && listenErr != http.ErrServerClosed {
			app.Logger.Errorf("HTTP server fatal error: %v", listenErr)
			close(p.exit)
		}
	}()

	return nil
}

func (p *program) Stop(s service.Service) error {
	app.Logger.Info("Stopping sql-proxy service...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := p.httpServer.Shutdown(ctx); err != nil {
		app.Logger.Errorf("Server shutdown error: %v", err)
		return err
	}

	select {
	case <-p.exit:
		// already closed
	default:
		close(p.exit)
	}

	app.Logger.Info("Service stopped gracefully")
	return nil
}

type config struct {
	BindAddress string
	BindPort    int
	TLSCert     string
	TLSKey      string
}

func loadConfig() (*config, error) {
	app.DebugLog = app.GetEnvBool("DEBUG_LOG", false)

	bindAddress := app.GetEnvString("BIND_ADDR", "localhost")
	if bindAddress == "*" {
		bindAddress = ""
	}

	bindPort := app.GetEnvInt("BIND_PORT", 8080)
	if bindPort < 1 || bindPort > 65535 {
		return nil, fmt.Errorf("invalid BIND_PORT: %d", bindPort)
	}

	db.MaxRows = uint32(app.GetEnvInt("MAX_ROWS", 10000))
	db.MaxConnections = app.GetEnvInt("MAX_CONNECTIONS", 100)

	tlsCert := app.GetEnvString("TLS_CERT", "")
	tlsKey := app.GetEnvString("TLS_KEY", "")

	if (tlsCert != "" && tlsKey == "") || (tlsCert == "" && tlsKey != "") {
		return nil, fmt.Errorf("TLS_CERT and TLS_KEY must be both set or both empty")
	}

	return &config{
		BindAddress: bindAddress,
		BindPort:    bindPort,
		TLSCert:     tlsCert,
		TLSKey:      tlsKey,
	}, nil
}

func main() {
	svcConfig := &service.Config{
		Name:         "sql-proxy",
		DisplayName:  "SQL Proxy Service",
		Description:  "A lightweight REST service designed to replace ADODB calls in legacy software systems",
		Arguments:    []string{},
		Dependencies: []string{"network.target"},
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		err := service.Control(s, os.Args[1])
		if err != nil {
			log.Fatalf("Service control error: %v", err)
		}
		return
	}

	if !service.Interactive() {
		if err := s.Run(); err != nil {
			log.Fatalf("Service run error: %v", err)
		}
	} else {
		fmt.Println("Running in console mode...")
		if err := prg.Start(nil); err != nil {
			log.Fatalf("Failed to start in console mode: %v", err)
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nReceived interrupt signal, shutting down...")
		if err := prg.Stop(nil); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}
}
