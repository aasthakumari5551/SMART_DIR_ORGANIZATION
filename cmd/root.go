package cmd

import (
	"fmt"
	"os"

	"github.com/smazmi/smartdir-proto/pkg/ai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "smartdir",
	Short: "Smart directory management system",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
		// Initialize AI client
		ai.Init()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Add configuration validation and defaults
func initConfig() {
	// Set defaults
	viper.SetDefault("database.path", "$HOME/.smartdir/smartdir.db")
	viper.SetDefault("search.host", "http://localhost:7700")
	viper.SetDefault("search.api_key", "")

	// Load config from paths
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.smartdir")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file not found, using defaults")
		} else {
			fmt.Printf("Error reading config: %v\n", err)
		}
	} else {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// Validate required settings
	if viper.GetString("groq.api_key") == "" {
		fmt.Println("Warning: Groq API key not set, AI categorization will use fallback method")
	}
}
