package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize smartdir configuration",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		configDir := filepath.Join(home, ".smartdir")

		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			return
		}

		// Initialize database
		if err := storage.InitDB(filepath.Join(configDir, "smartdir.db")); err != nil {
			fmt.Printf("Error initializing database: %v\n", err)
			return
		}

		fmt.Println("Smartdir initialized successfully")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
