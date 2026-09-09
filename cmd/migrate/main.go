package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang-base/config"
	"golang-base/internal/database"

	"github.com/uptrace/bun"
	"github.com/urfave/cli/v2"
)

func initDB() (*bun.DB, func(), error) {
	cfg := config.LoadConfig()

	if err := database.InitPostgres(cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return database.DB, func() { database.Close() }, nil
}

func getMigrationsDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	dir := filepath.Join(cwd, "migrations")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir, nil
	}

	parentDir := filepath.Join(cwd, "..", "migrations")
	if info, err := os.Stat(parentDir); err == nil && info.IsDir() {
		return filepath.Clean(parentDir), nil
	}

	return dir, nil
}

func main() {
	app := &cli.App{
		Name:  "migrate",
		Usage: "Database migration tool for golang-base",
		Commands: []*cli.Command{
			{
				Name:  "up",
				Usage: "Run all pending migrations",
				Action: func(c *cli.Context) error {
					db, cleanup, err := initDB()
					if err != nil {
						return err
					}
					defer cleanup()
					return runMigrations(db)
				},
			},
			{
				Name:  "down",
				Usage: "Rollback the last migration",
				Action: func(c *cli.Context) error {
					db, cleanup, err := initDB()
					if err != nil {
						return err
					}
					defer cleanup()
					return rollbackMigration(db)
				},
			},
			{
				Name:  "create",
				Usage: "Create a new migration file",
				Args:  true,
				Action: func(c *cli.Context) error {
					name := c.Args().First()
					if strings.TrimSpace(name) == "" {
						return fmt.Errorf("migration name is required. Usage: migrate create <name>")
					}
					return createMigrationFile(name)
				},
			},
			{
				Name:  "list",
				Usage: "List all migrations and their status",
				Action: func(c *cli.Context) error {
					db, cleanup, err := initDB()
					if err != nil {
						return err
					}
					defer cleanup()
					return listMigrations(db)
				},
			},
			{
				Name:  "fresh",
				Usage: "Drop all tables and re-run all migrations",
				Action: func(c *cli.Context) error {
					db, cleanup, err := initDB()
					if err != nil {
						return err
					}
					defer cleanup()
					return freshMigration(db)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// ensureMigrationsTable creates the migrations table if it doesn't exist
func ensureMigrationsTable(ctx context.Context, db *bun.DB) {
	_, _ = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migrations (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		batch INTEGER NOT NULL DEFAULT 0
	)`)
}

func runMigrations(db *bun.DB) error {
	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return err
	}

	// Use ReadDir instead of filepath.Glob
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			files = append(files, filepath.Join(migrationsDir, entry.Name()))
		}
	}

	if len(files) == 0 {
		fmt.Println("No migrations found at:", migrationsDir)
		return nil
	}

	// Sort files to ensure consistent order
	sort.Strings(files)

	ctx := context.Background()
	ensureMigrationsTable(ctx, db)

	// Get current batch number (max batch in table)
	var currentBatch int
	err = db.QueryRowContext(ctx, "SELECT COALESCE(MAX(batch), 0) FROM migrations").Scan(&currentBatch)
	if err != nil {
		return fmt.Errorf("failed to get current batch: %w", err)
	}
	nextBatch := currentBatch + 1

	for _, file := range files {
		migrationName := strings.TrimSuffix(filepath.Base(file), ".up.sql")

		// Check if already applied
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM migrations WHERE name = ?", migrationName).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if count > 0 {
			fmt.Printf("Already applied: %s\n", filepath.Base(file))
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		contentStr := strings.TrimSpace(string(content))
		if contentStr == "" {
			continue
		}

		_, err = db.ExecContext(ctx, contentStr)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		// Track migration in migrations with batch
		_, _ = db.ExecContext(ctx, "INSERT INTO migrations (name, batch) VALUES (?, ?)", migrationName, nextBatch)

		fmt.Printf("Ran migration: %s\n", filepath.Base(file))
	}

	fmt.Println("\nAll migrations completed successfully!")
	return nil
}

func rollbackMigration(db *bun.DB) error {
	ctx := context.Background()

	// Get the current batch number (highest batch)
	var currentBatch int
	err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(batch), 0) FROM migrations").Scan(&currentBatch)
	if err != nil {
		return fmt.Errorf("failed to get current batch: %w", err)
	}

	if currentBatch == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}

	// Get all migrations in the current batch (ordered by id desc for rollback)
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM migrations WHERE batch = ? ORDER BY id DESC", currentBatch)
	if err != nil {
		return fmt.Errorf("failed to get migrations: %w", err)
	}
	defer rows.Close()

	var migrations []struct {
		ID   int
		Name string
	}
	for rows.Next() {
		var m struct {
			ID   int
			Name string
		}
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return fmt.Errorf("failed to scan migration: %w", err)
		}
		migrations = append(migrations, m)
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations to rollback in batch", currentBatch)
		return nil
	}

	fmt.Printf("Rolling back batch %d (%d migrations)\n", currentBatch, len(migrations))

	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		downFile := filepath.Join(migrationsDir, m.Name+".down.sql")

		content, err := os.ReadFile(downFile)
		if err != nil {
			fmt.Printf("Failed to read rollback file %s: %v\n", downFile, err)
			continue
		}

		_, err = db.ExecContext(ctx, string(content))
		if err != nil {
			if !strings.Contains(err.Error(), "does not exist") {
				fmt.Printf("Failed to rollback %s: %v\n", m.Name, err)
				continue
			}
		}

		_, _ = db.ExecContext(ctx, "DELETE FROM migrations WHERE id = ?", m.ID)
		fmt.Printf("Rolled back: %s\n", m.Name)
	}

	fmt.Println("Batch rollback completed")
	return nil
}

func listMigrations(db *bun.DB) error {
	ctx := context.Background()

	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return err
	}

	// Read all migration files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	// Ensure migrations table exists
	ensureMigrationsTable(ctx, db)

	// Get applied migrations from database
	rows, err := db.QueryContext(ctx, "SELECT name, batch, applied_at FROM migrations ORDER BY id")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]struct {
		Batch     int
		AppliedAt string
	})
	for rows.Next() {
		var name string
		var batch int
		var appliedAt string
		if err := rows.Scan(&name, &batch, &appliedAt); err != nil {
			continue
		}
		applied[name] = struct {
			Batch     int
			AppliedAt string
		}{Batch: batch, AppliedAt: appliedAt}
	}

	fmt.Println("\nMigrations Status:")
	fmt.Println("=================")
	fmt.Printf("%-40s %-8s %-8s %s\n", "Name", "Status", "Batch", "Applied At")
	fmt.Println(strings.Repeat("-", 80))

	pendingCount := 0
	for _, file := range files {
		migrationName := strings.TrimSuffix(file, ".up.sql")
		if info, ok := applied[migrationName]; ok {
			fmt.Printf("%-40s %-8s %-8d %s\n", file, "applied", info.Batch, info.AppliedAt)
		} else {
			fmt.Printf("%-40s %-8s\n", file, "pending")
			pendingCount++
		}
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("\nTotal: %d | Applied: %d | Pending: %d\n", len(files), len(applied), pendingCount)

	return nil
}

func freshMigration(db *bun.DB) error {
	ctx := context.Background()

	fmt.Println("Dropping all tables...")
	if _, err := db.ExecContext(ctx, "DROP SCHEMA public CASCADE"); err != nil {
		return fmt.Errorf("failed to drop schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA public"); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	fmt.Println("Schema recreated, running all migrations...")
	return runMigrations(db)
}

func getNextMigrationSequence(migrationsDir string) int {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 1
	}

	maxSeq := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if strings.HasSuffix(filename, ".up.sql") || strings.HasSuffix(filename, "_up.sql") {
			parts := strings.SplitN(filename, "_", 2)
			if len(parts) > 1 {
				var seq int
				if _, err := fmt.Sscanf(parts[0], "%d", &seq); err == nil {
					if seq > maxSeq {
						maxSeq = seq
					}
				}
			}
		}
	}
	return maxSeq + 1
}

func createMigrationFile(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("migration name is required")
	}

	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ToLower(name)

	var migrationName string
	// Check if user already provided a numbered prefix e.g. "003_something"
	matched, _ := regexp.MatchString(`^\d{3,}_`, name)
	if matched {
		migrationName = name
	} else {
		nextSeq := getNextMigrationSequence(migrationsDir)
		migrationName = fmt.Sprintf("%03d_%s", nextSeq, name)
	}

	upFile := filepath.Join(migrationsDir, migrationName+".up.sql")
	downFile := filepath.Join(migrationsDir, migrationName+".down.sql")

	currentTime := time.Now().Format("2006-01-02 15:04:05")

	upContent := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n", migrationName, currentTime)
	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("failed to create up migration: %w", err)
	}
	fmt.Printf("Created: %s\n", upFile)

	downContent := fmt.Sprintf("-- Rollback: %s\n-- Created: %s\n\n", migrationName, currentTime)
	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("failed to create down migration: %w", err)
	}
	fmt.Printf("Created: %s\n", downFile)

	return nil
}
