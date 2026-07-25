package seeders

import (
	"fmt"
	"sort"
	"strings"

	"github.com/uptrace/bun"
)

// newUUID generates a UUID v7.
func newUUID() string {
	return ""
}

// Seeder interface — similar to Laravel's Database Seeder.
type Seeder interface {
	Run(db *bun.DB) error
	Name() string
	Order() int
}

// ArgSetter is an optional interface for seeders that accept arguments.
type ArgSetter interface {
	SetArgs(args []string)
}

// ForceMode bypasses count>0 skip check on all seeders
var ForceMode bool

// ForceCleanMode truncates and re-seeds
var ForceCleanMode bool

// ExcludeList holds names of seeders to skip
var ExcludeList []string

// registry holds all registered seeders.
var registry = make(map[string]Seeder)

// Register adds a seeder to the registry.
func Register(s Seeder) {
	name := s.Name()
	if _, ok := registry[name]; ok {
		panic(fmt.Sprintf("seeder %s already registered", name))
	}
	registry[name] = s
}

// RunByName runs a single seeder by its class name with optional args.
func RunByName(db *bun.DB, name string, args ...string) error {
	s, ok := registry[name]
	if !ok {
		var keys []string
		for k := range registry {
			keys = append(keys, k)
		}
		return fmt.Errorf("seeder %q not found. Available: %s", name, strings.Join(keys, ", "))
	}

	if argSetter, ok := s.(ArgSetter); ok {
		argSetter.SetArgs(args)
	}

	fmt.Printf("==> Running seeder: %s\n", name)
	if err := s.Run(db); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	fmt.Printf("==> Done: %s\n", name)
	return nil
}

// RunAll runs every registered seeder in ascending order.
func RunAll(db *bun.DB) error {
	names := Names()

	// Sort by Order() ascending
	sort.SliceStable(names, func(i, j int) bool {
		return registry[names[i]].Order() < registry[names[j]].Order()
	})

	excludeMap := make(map[string]bool)
	for _, e := range ExcludeList {
		excludeMap[e] = true
	}

	fmt.Printf("Running %d seeders...\n\n", len(names))
	for _, name := range names {
		s := registry[name]
		if excludeMap[name] {
			fmt.Printf("==> Skipping seeder: %s (Order: %d)\n\n", name, s.Order())
			continue
		}

		fmt.Printf("==> Running seeder: %s (Order: %d)\n", name, s.Order())
		if err := s.Run(db); err != nil {
			return fmt.Errorf("%s failed: %w", name, err)
		}
		fmt.Printf("==> Done: %s\n\n", name)
	}
	fmt.Println("All seeders completed successfully.")
	return nil
}

// Names returns all registered seeder names.
func Names() []string {
	var keys []string
	for k := range registry {
		keys = append(keys, k)
	}
	return keys
}
