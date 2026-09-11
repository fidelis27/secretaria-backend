package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port   int
	DBHost string
	DBPort int
	DBName string
	DBUser string
	DBPass string
	DBTLS  bool
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

	return Config{
		Port:   port,
		DBHost: os.Getenv("DB_HOST"),
		DBPort: dbPort,
		DBName: os.Getenv("DB_NAME"),
		DBUser: os.Getenv("DB_USER"),
		DBPass: os.Getenv("DB_PASSWORD"),
		DBTLS:  dbTLS,
	}, nil
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