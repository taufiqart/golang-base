package integration

import (
	"context"
	"fmt"
	"golang-base/cmd/seed/seeders"
	"golang-base/config"
	"golang-base/internal/app"
	"golang-base/internal/database"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

var (
	TestApp     *fiber.App
	projectRoot string
)

func TestMain(m *testing.M) {
	projectRoot = findProjectRoot()

	if e2eDB := os.Getenv("E2E_DATABASE_URL"); e2eDB != "" {
		os.Setenv("DATABASE_URL", e2eDB)
	}
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-secret")
	}

	os.Setenv("RATE_LIMIT_MAX", "1000000")
	cfg := config.LoadConfig()
	if cfg.DatabaseURL == "" {
		fmt.Println("E2E_DATABASE_URL not set. Skipping integration tests.")
		os.Exit(0)
	}

	if err := database.InitPostgres(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := freshMigrate(); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	TestApp = app.New(cfg)

	code := m.Run()
	os.Exit(code)
}

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func freshMigrate() error {
	ctx := context.Background()
	db := database.DB

	fmt.Println("Dropping all tables...")
	if _, err := db.ExecContext(ctx, "DROP SCHEMA IF EXISTS public CASCADE"); err != nil {
		return fmt.Errorf("drop schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA public"); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	fmt.Println("Running migrations...")
	migrationsDir := filepath.Join(projectRoot, "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migrations (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		batch INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".up.sql")

		var count int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM migrations WHERE name = ?", name).Scan(&count)
		if count > 0 {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}
		contentStr := strings.TrimSpace(string(content))
		if contentStr == "" {
			continue
		}

		if _, err := db.ExecContext(ctx, contentStr); err != nil {
			return fmt.Errorf("exec %s: %w", file, err)
		}

		var currentBatch int
		db.QueryRowContext(ctx, "SELECT COALESCE(MAX(batch), 0) FROM migrations").Scan(&currentBatch)
		db.ExecContext(ctx, "INSERT INTO migrations (name, batch) VALUES (?, ?)", name, currentBatch+1)

		fmt.Printf("  Ran: %s\n", filepath.Base(file))
	}

	fmt.Println("Migration complete!")
	return nil
}

func seedAdminUser(t *testing.T) {
	t.Helper()

	os.Setenv("SUPER_ADMIN_EMAIL", "admin@test.com")
	os.Setenv("SUPER_ADMIN_PASSWORD", "Test@1234")

	err := seeders.RunByName(database.DB, "RolePermissionsSeeder")
	if err != nil {
		t.Fatalf("seed role permissions: %v", err)
	}

	err = seeders.RunByName(database.DB, "SuperAdminSeeder")
	if err != nil {
		t.Fatalf("seed super admin: %v", err)
	}
}
