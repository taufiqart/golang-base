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
	_ "golang-base/internal/modules/auth"
	_ "golang-base/internal/modules/storage"
	_ "golang-base/internal/modules/user"
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
		case "list":
			fmt.Println("Available seeders:")
			allSeeders := seeders.GetAll()
			sort.Slice(allSeeders, func(i, j int) bool {
				return allSeeders[i].Order() < allSeeders[j].Order()
			})
			for _, s := range allSeeders {
				fmt.Printf("  [%d] %s\n", s.Order(), s.Name())
			}
			return
		}
	}

	// Run all registered seeders
	if err := seeders.RunAll(database.DB); err != nil {
		fmt.Fprintf(os.Stderr, "Seeder error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seeding completed successfully.")
}
