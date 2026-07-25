package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"golang-base/config"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// DB holds the database connection instance
var DB *bun.DB

// InitPostgres initializes PostgreSQL connection using Bun ORM
func InitPostgres(cfg *config.Config) error {
	if cfg.DatabaseURL == "" {
		log.Println("DATABASE_URL not configured, database will be disabled")
		return fmt.Errorf("DATABASE_URL not configured")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DatabaseURL)))
	sqldb.SetMaxIdleConns(10)
	sqldb.SetConnMaxLifetime(5 * time.Minute)
	db := bun.NewDB(sqldb, pgdialect.New())

	if err := db.Ping(); err != nil {
		log.Printf("Database connection failed: %v (Database will be disabled)", err)
		return err
	}

	log.Println("PostgreSQL connection successfully established")
	DB = db
	return nil
}

// Close closes the database connection
func Close() {
	if DB != nil {
		DB.Close()
	}
}
