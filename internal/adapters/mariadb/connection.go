package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/go-sql-driver/mysql"

	"github.com/karol/secretaria-escolar-backend/internal/platform/config"
)

type Connection struct {
	database *sql.DB
}

func Open(appConfig config.Config) (*Connection, error) {
	database, err := sql.Open("mysql", dataSourceName(appConfig))
	if err != nil {
		return nil, fmt.Errorf("open MariaDB connection: %w", err)
	}

	connection := &Connection{database: database}
	if err := connection.Check(context.Background()); err != nil {
		_ = database.Close()
		return nil, err
	}

	return connection, nil
}

func (connection *Connection) Check(ctx context.Context) error {
	if err := connection.database.PingContext(ctx); err != nil {
		return fmt.Errorf("ping MariaDB: %w", err)
	}
	return nil
}

func (connection *Connection) Close() error {
	return connection.database.Close()
}

func dataSourceName(appConfig config.Config) string {
	tls := "false"
	if appConfig.DBTLS {
		tls = "true"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&tls=%s",
		url.QueryEscape(appConfig.DBUser),
		url.QueryEscape(appConfig.DBPass),
		appConfig.DBHost,
		appConfig.DBPort,
		appConfig.DBName,
		tls,
	)
}
