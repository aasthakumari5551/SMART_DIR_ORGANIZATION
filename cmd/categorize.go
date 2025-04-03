package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smazmi/smartdir-proto/internal/core"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/cobra"
)

var categorizeCmd = &cobra.Command{
	Use:   "categorize [path]",
	Short: "Categorize files using AI",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := filepath.Join(os.Getenv("HOME"), ".smartdir", "smartdir.db")
		if err := storage.InitDB(dbPath); err != nil {
			fmt.Printf("Database initialization failed: %v\n", err)
			return
		}

		path := args[0]
		if err := core.CategorizeFiles(path); err != nil {
			fmt.Printf("Categorization error: %v\n", err)
			return
		}
		fmt.Println("Files categorized successfully")
	},
}

func init() {
	rootCmd.AddCommand(categorizeCmd)
}
