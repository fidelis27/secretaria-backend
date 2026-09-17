package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/fidelis27/secretaria-backend/internal/adapters/mariadb"
	"github.com/fidelis27/secretaria-backend/internal/platform/config"
	"github.com/fidelis27/secretaria-backend/migrations"
)

func main() {
	appConfig, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	database, err := mariadb.Open(appConfig)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := migrations.Apply(context.Background(), database.SQLDB()); err != nil {
		slog.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")
}
