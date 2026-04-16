package main

import (
	"log"
	"net/http"
	"time"

	"speedtest-exporter/internal/exporter"
)

func main() {
	config, err := exporter.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	handler := exporter.NewHandler(config, exporter.RunSpeedtest)
	server := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on :%s", config.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
