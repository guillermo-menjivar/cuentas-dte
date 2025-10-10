package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Config represents the application configuration
type Config struct {
	HaciendaURL        string `mapstructure:"hacienda_url"`
	Port               string `mapstructure:"port"`
	DatabaseURL        string `mapstructure:"database_url"`
	RedisURL           string `mapstructure:"redis_url"`
	VaultURL           string `mapstructure:"vault_url"`
	VaultToken         string `mapstructure:"vault_token"`
	VaultRetryAttempts int    `mapstructure:"vault_retry_attempts"`

	// Firmador configuration
	FirmadorURL          string        `mapstructure:"firmador_url"`
	FirmadorTimeout      time.Duration `mapstructure:"firmador_timeout"`
	FirmadorRetryMax     int           `mapstructure:"firmador_retry_max"`
	FirmadorRetryWaitMin time.Duration `mapstructure:"firmador_retry_wait_min"`
	FirmadorRetryWaitMax time.Duration `mapstructure:"firmador_retry_wait_max"`
}

var GlobalConfig Config

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "cuentas",
	Short: "Cuentas - DTE Management Application",
	Long: `Cuentas is a CLI application for managing DTE (Documentos Tributarios Electrónicos)
for El Salvador's Hacienda system.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Cuentas DTE Management System")
		fmt.Printf("Hacienda URL: %s\n", GlobalConfig.HaciendaURL)
		fmt.Printf("Firmador URL: %s\n", GlobalConfig.FirmadorURL)
		fmt.Printf("Server Port: %s\n", GlobalConfig.Port)
		fmt.Println("Use 'cuentas serve' to start the API server")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add subcommands
	RootCmd.AddCommand(ServeCmd)
	RootCmd.AddCommand(MigrateCmd)

	// Global flags
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cuentas.yaml)")

	// Bind environment variables
	viper.SetEnvPrefix("CUENTAS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("hacienda_url", "https://apitest.dtes.mh.gob.sv")
	viper.SetDefault("port", "8080")
	viper.SetDefault("database_url", "postgres://cuentas_user:cuentas_password@localhost:5432/cuentas?sslmode=disable")
	viper.SetDefault("redis_url", "redis://:redis_password@localhost:6379")
	viper.SetDefault("vault_url", "http://localhost:8200")
	viper.SetDefault("vault_token", "vault-root-token")
	viper.SetDefault("vault_retry_attempts", 10)

	// Firmador defaults
	viper.SetDefault("firmador_url", "http://localhost:8113/firmardocumento")
	viper.SetDefault("firmador_timeout", 30*time.Second)
	viper.SetDefault("firmador_retry_max", 3)
	viper.SetDefault("firmador_retry_wait_min", 1*time.Second)
	viper.SetDefault("firmador_retry_wait_max", 5*time.Second)
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
	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}
}
