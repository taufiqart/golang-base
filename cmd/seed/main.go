package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"golang-base/cmd/seed/seeders"
	_ "golang-base/cmd/seed/seeders"
	"golang-base/config"
	"golang-base/internal/database"
)

func main() {
	cfg := config.LoadConfig()

	if err := database.InitPostgres(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Filter out --force/-f and --force-clean flags before processing args
	remainingArgs := make([]string, 0, len(os.Args)-1)
	for _, a := range os.Args[1:] {
		if a == "--force" || a == "-f" {
			seeders.ForceMode = true
		} else if a == "--force-clean" || a == "-fc" {
			seeders.ForceCleanMode = true
		} else if strings.HasPrefix(a, "--except=") {
			parts := strings.Split(strings.TrimPrefix(a, "--except="), ",")
			seeders.ExcludeList = append(seeders.ExcludeList, parts...)
		} else {
			remainingArgs = append(remainingArgs, a)
		}
	}

	if len(remainingArgs) > 0 {
		switch strings.TrimSpace(remainingArgs[0]) {
		case "--list", "-l":
			names := seeders.Names()
			sort.Strings(names)
			fmt.Println("Available seeders:")
			for _, n := range names {
				fmt.Printf("  - %s\n", n)
			}
			return
		}

		name := strings.TrimSpace(remainingArgs[0])
		if name == "" {
			fmt.Fprintln(os.Stderr, "Usage: go run cmd/seed/main.go [SeederName] [args...]")
			fmt.Fprintln(os.Stderr, "  Example: go run cmd/seed/main.go CreateAdminSeeder admin@example.com password123")
			os.Exit(1)
		}
		var extraArgs []string
		if len(remainingArgs) > 1 {
			extraArgs = remainingArgs[1:]
		}
		if err := seeders.RunByName(database.DB, name, extraArgs...); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := seeders.RunAll(database.DB); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
