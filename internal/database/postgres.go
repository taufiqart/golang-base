package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"golang-base/config"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bunotel"
)

// DB holds the database connection instance
var DB *bun.DB

// SQL exposes the raw pool handle for connection-pool metrics, because bun.DB
// only reports query counters.
var SQL *sql.DB

// InitPostgres initializes PostgreSQL connection using Bun ORM
func InitPostgres(cfg *config.Config) error {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL not configured, database will be disabled")
		return fmt.Errorf("DATABASE_URL not configured")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DatabaseURL)))
	sqldb.SetMaxIdleConns(10)
	sqldb.SetConnMaxLifetime(5 * time.Minute)
	db := bun.NewDB(sqldb, pgdialect.New())

	// Formatted queries stay off: rendering bound values into a span would ship
	// row data and credentials to the trace backend.
	db.AddQueryHook(bunotel.NewQueryHook(bunotel.WithDBName(cfg.AppService)))

	if err := db.Ping(); err != nil {
		slog.Error("database connection failed, database will be disabled", "error", err)
		return err
	}

	slog.Info("postgresql connection established")
	DB = db
	SQL = sqldb
	return nil
}

// Close closes the database connection
func Close() {
	if DB != nil {
		DB.Close()
	}
}
