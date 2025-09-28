package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Config represents the application configuration
type Config struct {
	HaciendaURL string `mapstructure:"hacienda_url"`
}

var config Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cuentas",
	Short: "Cuentas - DTE Management Application",
	Long: `Cuentas is a CLI application for managing DTE (Documentos Tributarios Electrónicos)
for El Salvador's Hacienda system.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to Cuentas DTE Management System")
		fmt.Printf("Hacienda URL: %s\n", config.HaciendaURL)
	},
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

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cuentas.yaml)")

	// Bind environment variables
	viper.SetEnvPrefix("CUENTAS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("hacienda_url", "https://apitest.dtes.mh.gob.sv")
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
