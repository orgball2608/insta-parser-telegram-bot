package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/pressly/goose/v3"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate [up|down|status|reset|create <name>]")
	}

	command := os.Args[1]

	// The create command needs special handling
	if command == "create" {
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate create <name>")
		}
		name := os.Args[2]
		createMigration(name)
		return
	}

	// For all other commands, we need a database connection
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
	}

	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	migrationsDir := filepath.Join(wd, "migrations")
	fmt.Printf("Running migrations from: %s\n", migrationsDir)

	switch command {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations applied successfully")
	case "down":
		if err := goose.Down(db, migrationsDir); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		fmt.Println("Migration rollback successful")
	case "status":
		if err := goose.Status(db, migrationsDir); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
	case "reset":
		if err := goose.Reset(db, migrationsDir); err != nil {
			log.Fatalf("Failed to reset migrations: %v", err)
		}
		fmt.Println("All migrations have been rolled back")
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func createMigration(name string) {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	migrationsDir := filepath.Join(wd, "migrations")
	fmt.Printf("Creating migration in: %s\n", migrationsDir)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
	}

	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := goose.Create(db, migrationsDir, name, "sql"); err != nil {
		log.Fatalf("Failed to create migration: %v", err)
	}
} 