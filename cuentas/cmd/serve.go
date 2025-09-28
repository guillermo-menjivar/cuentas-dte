package cmd

import (
	"fmt"
	"log"

	"cuentas/internal/handlers"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var vaultService *services.VaultService

// ServeCmd represents the serve command
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  `Start the Cuentas API server with all endpoints.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting Cuentas API Server...")

		// Initialize Vault (required for API server)
		if err := initializeVault(); err != nil {
			log.Fatalf("Failed to initialize Vault: %v", err)
		}

		fmt.Printf("Server running on port: %s\n", GlobalConfig.Port)
		startServer()
	},
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

func startServer() {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.Default()

	// Add middleware to inject Vault service
	r.Use(func(c *gin.Context) {
		c.Set("vaultService", vaultService)
		c.Next()
	})

	// API v1 routes
	v1 := r.Group("/v1")
	{
		v1.GET("/health", handlers.HealthHandler)
	}

	// Start server
	port := ":" + GlobalConfig.Port
	fmt.Printf("Server starting on http://localhost%s\n", port)
	log.Fatal(r.Run(port))
}
