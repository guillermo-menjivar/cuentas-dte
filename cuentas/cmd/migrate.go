package cmd

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// MigrateCmd represents the migrate command
var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
	Long:  `Run database migrations for the Cuentas application.`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	Long:  `Run all pending database migrations.`,
	Run: func(cmd *cobra.Command, args []string) {
		runMigrations("up")
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the last migration",
	Long:  `Rollback the last database migration.`,
	Run: func(cmd *cobra.Command, args []string) {
		runMigrations("down")
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  `Show the current status of database migrations.`,
	Run: func(cmd *cobra.Command, args []string) {
		showMigrationStatus()
	},
}

func runMigrations(direction string) {
	databaseURL := viper.GetString("database_url")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set. Please set it in config.yaml or as CUENTAS_DATABASE_URL environment variable")
	}

	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize migrate: %v", err)
	}
	defer m.Close()

	switch direction {
	case "up":
		fmt.Println("Running migrations up...")
		if err := m.Up(); err != nil {
			if err == migrate.ErrNoChange {
				fmt.Println("No migrations to run")
				return
			}
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		fmt.Println("Rolling back last migration...")
		if err := m.Steps(-1); err != nil {
			if err == migrate.ErrNoChange {
				fmt.Println("No migrations to rollback")
				return
			}
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		fmt.Println("Migration rollback completed successfully")
	}
}

func showMigrationStatus() {
	databaseURL := viper.GetString("database_url")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set. Please set it in config.yaml or as CUENTAS_DATABASE_URL environment variable")
	}

	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize migrate: %v", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			fmt.Println("No migrations have been run")
			return
		}
		log.Fatalf("Failed to get migration version: %v", err)
	}

	fmt.Printf("Current migration version: %d\n", version)
	fmt.Printf("Database is dirty: %t\n", dirty)
}

func init() {
	MigrateCmd.AddCommand(migrateUpCmd)
	MigrateCmd.AddCommand(migrateDownCmd)
	MigrateCmd.AddCommand(migrateStatusCmd)
}
