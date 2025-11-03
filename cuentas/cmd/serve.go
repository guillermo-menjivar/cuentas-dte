package cmd

import (
	"fmt"
	"log"
	"time"

	"cuentas/internal/database"
	"cuentas/internal/dte"
	"cuentas/internal/hacienda"
	"cuentas/internal/handlers"
	"cuentas/internal/middleware"
	"cuentas/internal/services"
	"cuentas/internal/services/firmador"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	vaultService    *services.VaultService
	haciendaService *services.HaciendaService // ⭐ Add this!
	firmadorClient  *firmador.Client
	haciendaClient  *hacienda.Client
	dteService      *dte.DTEService
)

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

		// Initialize Redis connection
		if err := initializeRedis(); err != nil {
			log.Fatalf("Failed to initialize Redis: %v", err)
		}
		defer database.CloseRedis()

		// Initialize Vault (required for API server)
		if err := initializeVault(); err != nil {
			log.Fatalf("Failed to initialize Vault: %v", err)
		}

		// Run database migrations automatically
		if err := runDatabaseMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		// Initialize Firmador client
		if err := initializeFirmador(); err != nil {
			log.Fatalf("Failed to initialize Firmador: %v", err)
		}

		// Initialize Hacienda client
		if err := initializeHacienda(); err != nil {
			log.Fatalf("Failed to initialize Hacienda: %v", err)
		}

		// Initialize Hacienda service (authentication & token caching) ⭐ Add this!
		if err := initializeHaciendaService(); err != nil {
			log.Fatalf("Failed to initialize Hacienda service: %v", err)
		}

		// Initialize DTE service
		if err := initializeDTEService(); err != nil {
			log.Fatalf("Failed to initialize DTE service: %v", err)
		}

		fmt.Printf("Server running on port: %s\n", GlobalConfig.Port)
		startServer()
	},
}

func initializeHacienda() error {
	fmt.Println("Initializing Hacienda client...")

	// Temporary fix - explicitly set the URL
	cfg := &hacienda.Config{
		BaseURL:      "https://apitest.dtes.mh.gob.sv/fesv/recepciondte",
		Timeout:      60 * time.Second,
		RetryMax:     3,
		RetryWaitMin: 2 * time.Second,
		RetryWaitMax: 10 * time.Second,
	}

	haciendaClient = hacienda.NewClient(cfg)

	fmt.Printf("Hacienda client initialized (URL: %s)\n", haciendaClient.GetBaseURL())
	return nil
}

// ⭐ Add this function!
func initializeHaciendaService() error {
	fmt.Println("Initializing Hacienda service...")

	// Create hacienda service (handles authentication & token caching)
	hs, err := services.NewHaciendaService(
		database.DB,
		vaultService,
		database.RedisClient,
	)
	if err != nil {
		return fmt.Errorf("failed to create Hacienda service: %v", err)
	}

	haciendaService = hs
	fmt.Println("Hacienda service initialized")
	return nil
}

func initializeDTEService() error {
	fmt.Println("Initializing DTE service...")

	dteService = dte.NewDTEService(
		database.DB,
		database.RedisClient,
		firmadorClient,
		vaultService,
		haciendaClient,
		haciendaService, // ⭐ Pass it here!
	)

	fmt.Println("DTE service initialized")
	return nil
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

func initializeRedis() error {
	fmt.Println("Connecting to Redis...")

	if err := database.InitRedis(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	fmt.Println("Successfully connected to Redis")
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

func initializeFirmador() error {
	fmt.Println("Initializing Firmador client...")

	// Create firmador client from viper config
	firmadorClient = firmador.NewClientFromViper()

	fmt.Printf("Firmador client initialized (URL: %s)\n", firmadorClient.GetBaseURL())
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

	// Add middleware to inject database, Redis, and Vault service
	r.Use(func(c *gin.Context) {
		c.Set("db", database.DB)             // Inject database connection
		c.Set("redis", database.RedisClient) // Inject Redis client
		c.Set("vaultService", vaultService)  // Inject Vault service
		c.Set("firmador", firmadorClient)
		c.Set("dteService", dteService)
		c.Next()
	})

	r.Use(middleware.CompanyIDMiddleware())

	// API v1 routes
	v1 := r.Group("/v1")
	{
		v1.GET("/health", handlers.HealthHandler)
		v1.POST("/companies", handlers.CreateCompanyHandler)
		v1.GET("/companies", handlers.ListCompaniesHandler)
		v1.GET("/companies/:id", handlers.GetCompanyHandler)
		v1.POST("/companies/:id/authenticate", handlers.AuthenticateCompanyHandler)

		v1.POST("/clients", handlers.CreateClientHandler)
		v1.GET("/clients/:id", handlers.GetClientHandler)
		v1.GET("/clients", handlers.ListClientsHandler)
		v1.PUT("/clients/:id", handlers.UpdateClientHandler)
		v1.DELETE("/clients/:id", handlers.DeleteClientHandler)

		// Inventory item routes
		inventorySvc := services.NewInventoryService(database.DB)
		inventoryHandler := handlers.NewInventoryHandler(inventorySvc)

		v1.POST("/inventory/items", inventoryHandler.CreateInventoryItemHandler)
		v1.GET("/inventory/items/:id", inventoryHandler.GetInventoryItemHandler)
		v1.GET("/inventory/items", inventoryHandler.ListInventoryItemsHandler)
		v1.PUT("/inventory/items/:id", inventoryHandler.UpdateInventoryItemHandler)
		v1.DELETE("/inventory/items/:id", inventoryHandler.DeleteInventoryItemHandler)

		// Inventory tax routes
		v1.GET("/inventory/items/:id/taxes", inventoryHandler.GetItemTaxesHandler)
		v1.POST("/inventory/items/:id/taxes", inventoryHandler.AddItemTaxHandler)
		v1.DELETE("/inventory/items/:id/taxes/:code", inventoryHandler.RemoveItemTaxHandler)

		// Inventory cost tracking (CQRS)
		v1.POST("/inventory/items/:id/purchase", inventoryHandler.RecordPurchaseHandler)
		v1.POST("/inventory/items/:id/adjustment", inventoryHandler.RecordAdjustmentHandler)
		v1.GET("/inventory/items/:id/state", inventoryHandler.GetInventoryStateHandler)
		v1.GET("/inventory/states", inventoryHandler.ListInventoryStatesHandler)
		v1.GET("/inventory/items/:id/cost-history", inventoryHandler.GetCostHistoryHandler)
		v1.GET("/inventory/items/:id/legal-register", inventoryHandler.GetLegalInventoryRegisterHandler)

		v1.GET("/inventory/events", inventoryHandler.GetAllEventsHandler)             // All events across all items
		v1.GET("/inventory/valuation", inventoryHandler.GetInventoryValuationHandler) // Point-in-time valuation

		// Invoice routes
		invoiceService := services.NewInvoiceService(inventorySvc)

		invoiceHandler := handlers.NewInvoiceHandler(invoiceService)
		v1.POST("/invoices", invoiceHandler.CreateInvoice)
		v1.GET("/invoices", invoiceHandler.ListInvoices)
		v1.GET("/invoices/:id", invoiceHandler.GetInvoice)
		v1.DELETE("/invoices/:id", invoiceHandler.DeleteInvoice)
		v1.POST("/invoices/:id/finalize", invoiceHandler.FinalizeInvoice)

		actividadHandler := handlers.NewActividadEconomicaHandler()
		v1.GET("/actividades-economicas/categories", actividadHandler.GetCategories)
		v1.GET("/actividades-economicas/categories/:code", actividadHandler.GetCategoryByCode)
		v1.GET("/actividades-economicas/categories/:code/activities", actividadHandler.GetActivitiesByCategory)
		v1.GET("/actividades-economicas/search", actividadHandler.SearchActivities)
		v1.GET("/actividades-economicas/:code", actividadHandler.GetActivityDetails)

		establishmentHandler := handlers.NewEstablishmentHandler()

		v1.POST("/establishments", establishmentHandler.CreateEstablishment)
		v1.GET("/establishments", establishmentHandler.ListEstablishments)
		v1.GET("/establishments/:id", establishmentHandler.GetEstablishment)
		v1.PATCH("/establishments/:id", establishmentHandler.UpdateEstablishment)
		v1.DELETE("/establishments/:id", establishmentHandler.DeactivateEstablishment)

		// Point of Sale routes
		v1.POST("/establishments/:id/pos", establishmentHandler.CreatePointOfSale)
		v1.GET("/establishments/:id/pos", establishmentHandler.ListPointsOfSale)
		v1.GET("/pos/:id", establishmentHandler.GetPointOfSale)
		v1.PATCH("/pos/:id", establishmentHandler.UpdatePointOfSale)
		v1.PATCH("/pos/:id/location", establishmentHandler.UpdatePOSLocation)
		v1.DELETE("/pos/:id", establishmentHandler.DeactivatePointOfSale)
		// commitlog
		v1.GET("/dte/commit-log", handlers.ListDTECommitLogHandler)
		v1.GET("/dte/commit-log/:codigo_generacion", handlers.GetDTECommitLogEntryHandler)
		// notas

		notaService := services.NewNotaService()
		notasHandler := handlers.NewNotaHandler(notaService, invoiceService)

		notas := v1.Group("/notas")
		{
			// Nota de Débito
			notas.POST("/debito", notasHandler.CreateNotaDebito)
			notas.GET("/debito/:id", notasHandler.GetNotaDebito)

			// Nota de Crédito
			//notas.POST("/credito", notaHandler.CreateNotaCredito)
			//notas.POST("/credito/:id/finalize", notaHandler.FinalizeNotaCredito)

			// Common operations
			//notas.GET("/:id", notaHandler.GetNota)
			//notas.GET("", notaHandler.ListNotas)
			//notas.DELETE("/:id", notaHandler.DeleteNota)
		}

		reconciliationService := services.NewDTEReconciliationService(
			database.DB,
			haciendaClient,
			haciendaService,
		)
		reconciliationHandler := handlers.NewDTEReconciliationHandler(reconciliationService)

		v1.GET("/dte/reconciliation", reconciliationHandler.ReconcileDTEs)
		v1.GET("/dte/reconciliation/:codigo_generacion", reconciliationHandler.ReconcileSingleDTE)
	}

	// Start server
	port := ":" + GlobalConfig.Port
	fmt.Printf("Server starting on http://localhost%s\n", port)
	log.Fatal(r.Run(port))
}
