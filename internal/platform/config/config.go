package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                     int
	AppEnv                   string
	DBHost                   string
	DBPort                   int
	DBName                   string
	DBUser                   string
	DBPass                   string
	DBTLS                    bool
	DBTLSSkipVerify          bool
	OIDCIssuer               string
	OIDCAudience             string
	BootstrapSuperAdminEmail string
}

func Load() (Config, error) {
	port, err := envInt("PORT", 3333)
	if err != nil {
		return Config{}, err
	}

	dbPort, err := envInt("DB_PORT", 3306)
	if err != nil {
		return Config{}, err
	}

	dbTLS, err := envBool("DB_TLS", false)
	if err != nil {
		return Config{}, err
	}
	dbTLSSkipVerify, err := envBool("DB_TLS_SKIP_VERIFY", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:                     port,
		AppEnv:                   envString("APP_ENV", "development"),
		DBHost:                   os.Getenv("DB_HOST"),
		DBPort:                   dbPort,
		DBName:                   os.Getenv("DB_NAME"),
		DBUser:                   os.Getenv("DB_USER"),
		DBPass:                   os.Getenv("DB_PASSWORD"),
		DBTLS:                    dbTLS,
		DBTLSSkipVerify:          dbTLSSkipVerify,
		OIDCIssuer:               os.Getenv("OIDC_ISSUER"),
		OIDCAudience:             envString("OIDC_AUDIENCE", "authenticated"),
		BootstrapSuperAdminEmail: strings.TrimSpace(os.Getenv("BOOTSTRAP_SUPER_ADMIN_EMAIL")),
	}, nil
}

func envString(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return parsed, nil
}

func envBool(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be boolean: %w", name, err)
	}
	return parsed, nil
}
