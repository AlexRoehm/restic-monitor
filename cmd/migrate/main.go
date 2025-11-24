package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	dbPath := flag.String("db", "data/restic-monitor.db", "Path to SQLite database")
	tenantID := flag.String("tenant", "", "Tenant ID (optional, generates new if empty)")
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("Database path is required")
	}

	// Parse or generate tenant ID
	var tid uuid.UUID
	var err error
	if *tenantID != "" {
		tid, err = uuid.Parse(*tenantID)
		if err != nil {
			log.Fatalf("Invalid tenant ID: %v", err)
		}
	} else {
		tid = uuid.New()
		fmt.Printf("Generated new tenant ID: %s\n", tid)
	}

	// Open database
	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	fmt.Printf("Running migrations on database: %s\n", *dbPath)
	fmt.Printf("Tenant ID: %s\n\n", tid)

	// Initialize migration runner
	ctx := context.Background()
	runner := store.NewMigrationRunner(db)

	if err := runner.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize migration runner: %v", err)
	}

	// Get all migrations
	migrations := store.GetAllMigrations(tid)

	fmt.Printf("Found %d migrations\n", len(migrations))

	// Run migrations
	for _, migration := range migrations {
		applied, err := runner.IsApplied(ctx, migration.Version)
		if err != nil {
			log.Fatalf("Failed to check migration status: %v", err)
		}

		if applied {
			fmt.Printf("✓ Migration %s already applied: %s\n", migration.Version, migration.Description)
			continue
		}

		fmt.Printf("→ Running migration %s: %s\n", migration.Version, migration.Description)

		if err := runner.Run(ctx, migration); err != nil {
			log.Fatalf("Failed to run migration %s: %v", migration.Version, err)
		}

		fmt.Printf("✓ Migration %s completed\n", migration.Version)
	}

	fmt.Println("\nAll migrations completed successfully!")

	// Show migration status
	var applied []store.SchemaMigration
	if err := db.Order("version asc").Find(&applied).Error; err != nil {
		log.Fatalf("Failed to query migration status: %v", err)
	}

	fmt.Println("\nApplied migrations:")
	for _, m := range applied {
		fmt.Printf("  %s - %s (applied at %s)\n", m.Version, m.Description, m.AppliedAt.Format("2006-01-02 15:04:05"))
	}

	os.Exit(0)
}
