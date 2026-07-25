package main

import (
	"fmt"
	"log"

	"golang-base/config"
	"golang-base/internal/app"
	"golang-base/internal/database"
)

func main() {
	// 1. Load configuration
	cfg := config.LoadConfig()

	// 2. Initialize database connections
	db := database.InitPostgres(cfg)
	if db != nil {
		defer database.Close()
	}

	redis := database.InitRedis(cfg)
	if redis != nil {
		defer database.CloseRedis()
	}

	// 3. Create and start Fiber app
	application := app.New(cfg)

	log.Printf("Starting server on port %s...", cfg.Port)
	if err := application.Listen(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
