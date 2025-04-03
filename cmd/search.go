package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/meilisearch/meilisearch-go"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/cobra"

	"github.com/smazmi/smartdir-proto/models"
	"github.com/spf13/viper"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search files using natural language",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := filepath.Join(os.Getenv("HOME"), ".smartdir", "smartdir.db")
		if err := storage.InitDB(dbPath); err != nil {
			fmt.Printf("Database initialization failed: %v\n", err)
			return
		}
		client := meilisearch.New(viper.GetString("search.host"),
			meilisearch.WithAPIKey(viper.GetString("search.api_key")),
		)

		index := client.Index("files")

		// Perform search with empty search request parameters
		results, err := index.Search(args[0], &meilisearch.SearchRequest{})
		if err != nil {
			log.Fatalf("Search error: %v\n", err)
		}

		// Unmarshal results into structured format
		var files []models.FileDocument
		rawHits, _ := json.Marshal(results.Hits)
		if err := json.Unmarshal(rawHits, &files); err != nil {
			log.Fatalf("Failed to parse results: %v", err)
		}

		if len(files) == 0 {
			fmt.Println("No results found")
			return
		}

		// Add to the search.go file before displaying results:
		for _, file := range files {
			// Record file access in the database
			var access models.FileAccess
			result := storage.DB.Where("file_path = ?", file.Path).First(&access)
			if result.Error != nil {
				// Create new access record if not exists
				access = models.FileAccess{
					FilePath:    file.Path,
					AccessCount: 1,
				}
				storage.DB.Create(&access)
			} else {
				// Update existing access count
				storage.DB.Model(&access).Update("access_count", access.AccessCount+1)
			}
		}

		fmt.Println("Search results:")
		for _, file := range files {
			fmt.Printf("- %s\n  Category: %s\n  ID: %d\n\n", file.Path, file.Category, file.ID)
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
