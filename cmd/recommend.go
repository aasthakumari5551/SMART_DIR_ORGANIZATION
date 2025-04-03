package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/smazmi/smartdir-proto/models"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	limit     int
	timeRange int
)

var recommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "Get file recommendations based on access patterns",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := filepath.Join(os.Getenv("HOME"), ".smartdir", "smartdir.db")
		if err := storage.InitDB(dbPath); err != nil {
			fmt.Printf("Database initialization failed: %v\n", err)
			return
		}

		// Get frequently accessed files
		var accessStats []struct {
			FilePath    string
			AccessCount int
			Category    string
		}

		since := time.Now().AddDate(0, 0, -timeRange) // Last N days

		result := storage.DB.Table("file_accesses").
			Select("file_accesses.file_path, file_accesses.access_count, files.category").
			Joins("LEFT JOIN files ON files.path = file_accesses.file_path").
			Where("file_accesses.updated_at > ?", since).
			Order("file_accesses.access_count DESC").
			Limit(limit).
			Find(&accessStats)

		if result.Error != nil {
			fmt.Printf("Error fetching recommendations: %v\n", result.Error)
			return
		}

		if len(accessStats) == 0 {
			fmt.Println("No file access history available yet.")
			fmt.Println("Try using the 'search' command more to build up access statistics.")
			return
		}

		fmt.Println("Frequently accessed files:")
		for i, stat := range accessStats {
			// Format the path - use relative path if possible
			home, _ := os.UserHomeDir()
			displayPath := stat.FilePath
			if strings.HasPrefix(displayPath, home) {
				displayPath = "~" + displayPath[len(home):]
			}

			fmt.Printf("%d. %s (%s, accessed %d times)\n",
				i+1, displayPath, stat.Category, stat.AccessCount)
		}

		// Also recommend based on similar categories/tags
		fmt.Println("\nRelated files you might be interested in:")
		recommendSimilarFiles()
	},
}

func init() {
	rootCmd.AddCommand(recommendCmd)
	recommendCmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of recommendations")
	recommendCmd.Flags().IntVar(&timeRange, "days", 30, "Consider files accessed in the last N days")
}

func recommendSimilarFiles() {
	// Get the most recently accessed files
	var recentFiles []models.File
	storage.DB.Table("files").
		Joins("JOIN file_accesses ON files.path = file_accesses.file_path").
		Order("file_accesses.updated_at DESC").
		Limit(5).
		Find(&recentFiles)

	if len(recentFiles) == 0 {
		return
	}

	// Find files with similar tags or in the same category
	recentCategories := make(map[string]bool)
	recentTags := make(map[string]bool)

	for _, file := range recentFiles {
		recentCategories[file.Category] = true
		for _, tag := range strings.Split(file.Tags, ",") {
			if tag != "" {
				recentTags[tag] = true
			}
		}
	}

	var recommendations []models.File
	query := storage.DB.Table("files")

	// Exclude recently accessed files
	for _, file := range recentFiles {
		query = query.Where("path != ?", file.Path)
	}

	// Find similar files
	if len(recentTags) > 0 {
		for tag := range recentTags {
			query = query.Or("tags LIKE ?", "%"+tag+"%")
		}
	} else if len(recentCategories) > 0 {
		for category := range recentCategories {
			query = query.Or("category = ?", category)
		}
	}

	query.Limit(5).Find(&recommendations)

	for i, rec := range recommendations {
		fmt.Printf("%d. %s (%s)\n", i+1, filepath.Base(rec.Path), rec.Category)
	}
}
