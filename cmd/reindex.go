package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/meilisearch/meilisearch-go"
	"github.com/smazmi/smartdir-proto/models"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Rebuild the search index with all file metadata",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := filepath.Join(os.Getenv("HOME"), ".smartdir", "smartdir.db")
		if err := storage.InitDB(dbPath); err != nil {
			fmt.Printf("Database initialization failed: %v\n", err)
			return
		}

		// Get all files from the database
		var files []models.File
		result := storage.DB.Find(&files)
		if result.Error != nil {
			fmt.Printf("Error fetching files: %v\n", result.Error)
			return
		}

		if len(files) == 0 {
			fmt.Println("No files found in database")
			return
		}

		fmt.Printf("Reindexing %d files...\n", len(files))

		// Initialize Meilisearch client
		client := meilisearch.New(viper.GetString("search.host"),
			meilisearch.WithAPIKey(viper.GetString("search.api_key")),
		)

		// Get the index
		index := client.Index("files")

		// Configure searchable attributes
		settingsTask, err := index.UpdateSettings(&meilisearch.Settings{
			SearchableAttributes: []string{"path", "category", "tags"},
		})
		if err != nil {
			fmt.Printf("Warning: Failed to update searchable attributes: %v\n", err)
		} else {
			// Wait for the settings to be applied
			client.WaitForTask(settingsTask.TaskUID, 30)
		}

		// Prepare documents for Meilisearch
		docs := make([]map[string]any, 0, len(files))
		for _, file := range files {
			doc := map[string]any{
				"id":       file.ID,
				"path":     file.Path,
				"category": file.Category,
				"hash":     file.Hash,
				"tags":     file.Tags,
			}
			docs = append(docs, doc)
		}

		// Add all documents to search index
		task, err := index.UpdateDocuments(docs, "id")
		if err != nil {
			fmt.Printf("Failed to update index: %v\n", err)
			return
		}

		// Wait for indexing to complete
		client.WaitForTask(task.TaskUID, 60) // 60 seconds timeout

		fmt.Println("Search index rebuilt successfully")
	},
}

func init() {
	rootCmd.AddCommand(reindexCmd)
}
