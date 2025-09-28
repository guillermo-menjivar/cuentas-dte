package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"cuentas/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Config represents the application configuration
type Config struct {
	HaciendaURL string `mapstructure:"hacienda_url"`
	Port        string `mapstructure:"port"`
}

var config Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cuentas",
	Short: "Cuentas - DTE Management Application",
	Long: `Cuentas is a CLI application for managing DTE (Documentos Tributarios Electrónicos)
for El Salvador's Hacienda system.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting Cuentas DTE Management System")
		fmt.Printf("Hacienda URL: %s\n", config.HaciendaURL)
		fmt.Printf("Server Port: %s\n", config.Port)
		startServer()
	},
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  `Start the Cuentas API server with all endpoints.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting Cuentas API Server...")
		fmt.Printf("Server running on port: %s\n", config.Port)
		startServer()
	},
}

func startServer() {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.Default()

	// API v1 routes
	v1 := r.Group("/v1")
	{
		v1.GET("/health", handlers.HealthHandler)
	}

	// Start server
	port := ":" + config.Port
	fmt.Printf("Server starting on http://localhost%s\n", port)
	log.Fatal(r.Run(port))
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add subcommands
	rootCmd.AddCommand(serveCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cuentas.yaml)")

	// Bind environment variables
	viper.SetEnvPrefix("CUENTAS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("hacienda_url", "https://apitest.dtes.mh.gob.sv")
	viper.SetDefault("port", "8080")
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cuentas" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cuentas")
		viper.SetConfigName("config")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Unmarshal config into struct
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}
}

func main() {
	Execute()
}
