package main

import (
	"fmt"
	"log"

	"github.com/karol/secretaria-escolar-backend/internal/adapters/httpserver"
	"github.com/karol/secretaria-escolar-backend/internal/application/health"
	"github.com/karol/secretaria-escolar-backend/internal/platform/config"
)

func main() {
	appConfig, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	address := fmt.Sprintf("0.0.0.0:%d", appConfig.Port)
	server := httpserver.New(address, health.NewService())
	log.Printf("server listening on %s", address)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
