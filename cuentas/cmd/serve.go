package cmd

import (
	"fmt"
	"log"

	"cuentas/internal/database"
	"cuentas/internal/handlers"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var vaultService *services.VaultService

// ServeCmd represents the serve command
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  `Start the Cuentas API server with all endpoints.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting Cuentas API Server...")

		// Initialize database connection
		if err := initializeDatabase(); err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		defer database.CloseDB()

		// Initialize Vault (required for API server)
		if err := initializeVault(); err != nil {
			log.Fatalf("Failed to initialize Vault: %v", err)
		}

		// Run database migrations automatically
		if err := runDatabaseMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		fmt.Printf("Server running on port: %s\n", GlobalConfig.Port)
		startServer()
	},
}

func initializeDatabase() error {
	fmt.Println("Connecting to database...")

	if err := database.InitDB(); err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Test the connection
	if err := database.DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("Successfully connected to database")
	return nil
}

func initializeVault() error {
	fmt.Println("Connecting to Vault...")

	// Wait for Vault to be available (required)
	if err := services.WaitForVault(GlobalConfig.VaultRetryAttempts); err != nil {
		return fmt.Errorf("vault is required but unavailable: %v", err)
	}

	// Create Vault service
	vs, err := services.NewVaultService()
	if err != nil {
		return fmt.Errorf("failed to create Vault service: %v", err)
	}

	vaultService = vs
	fmt.Println("Successfully connected to Vault")

	return nil
}

func runDatabaseMigrations() error {
	fmt.Println("Running database migrations...")

	databaseURL := viper.GetString("database_url")
	if databaseURL == "" {
		return fmt.Errorf("database_url is not set")
	}

	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			fmt.Println("No migrations to run")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	fmt.Println("Database migrations completed successfully")
	return nil
}

func startServer() {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.Default()

	// Add middleware to inject database and Vault service
	r.Use(func(c *gin.Context) {
		c.Set("db", database.DB)            // Inject database connection
		c.Set("vaultService", vaultService) // Inject Vault service
		c.Next()
	})

	// API v1 routes
	v1 := r.Group("/v1")
	{
		v1.GET("/health", handlers.HealthHandler)
		v1.POST("/companies", handlers.CreateCompanyHandler)
		v1.GET("/companies/:id", handlers.GetCompanyHandler)
	}

	// Start server
	port := ":" + GlobalConfig.Port
	fmt.Printf("Server starting on http://localhost%s\n", port)
	log.Fatal(r.Run(port))
}
