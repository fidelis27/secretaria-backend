package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/fidelis27/secretaria-backend/internal/adapters/mariadb"
	applicationbootstrap "github.com/fidelis27/secretaria-backend/internal/application/bootstrap"
	"github.com/fidelis27/secretaria-backend/internal/platform/config"
	"github.com/fidelis27/secretaria-backend/migrations"
	devseeds "github.com/fidelis27/secretaria-backend/seeds/dev"
)

func main() {
	appConfig, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	if strings.EqualFold(appConfig.AppEnv, devseeds.ProductionEnv) {
		slog.Error("refusing to run dev seeds in production")
		os.Exit(1)
	}

	database, err := mariadb.Open(appConfig)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	ctx := context.Background()
	if err := migrations.Apply(ctx, database.SQLDB()); err != nil {
		slog.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}
	if err := devseeds.Apply(ctx, database.SQLDB(), appConfig.AppEnv); err != nil {
		slog.Error("failed to apply dev seeds", "error", err)
		os.Exit(1)
	}

	bootstrapService := applicationbootstrap.NewService(mariadb.NewBootstrapRepository(database))
	if _, err := bootstrapService.EnsureFirstSuperAdmin(ctx, appConfig.BootstrapSuperAdminEmail); err != nil {
		slog.Error("failed to bootstrap first superadmin", "error", err)
		os.Exit(1)
	}

	slog.Info("dev seeds applied")
}
