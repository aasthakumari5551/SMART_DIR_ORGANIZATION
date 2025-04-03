// cmd/stats.go
package cmd

import (
	"fmt"

	"github.com/smazmi/smartdir-proto/models"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show statistics about categorized files",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := filepath.Join(os.Getenv("HOME"), ".smartdir", "smartdir.db")
		if err := storage.InitDB(dbPath); err != nil {
			fmt.Printf("Database initialization failed: %v\n", err)
			return
		}

		var counts []struct {
			Category string
			Count    int64
		}

		result := storage.DB.Model(&models.File{}).
			Select("category, count(*) as count").
			Group("category").
			Find(&counts)

		if result.Error != nil {
			fmt.Printf("Error fetching statistics: %v\n", result.Error)
			return
		}

		fmt.Println("File Statistics:")
		var total int64
		for _, c := range counts {
			fmt.Printf("%-12s: %d files\n", c.Category, c.Count)
			total += c.Count
		}
		fmt.Printf("\nTotal files: %d\n", total)
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
