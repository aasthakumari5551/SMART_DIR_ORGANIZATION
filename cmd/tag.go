package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/meilisearch/meilisearch-go"
	"github.com/smazmi/smartdir-proto/models"
	"github.com/smazmi/smartdir-proto/pkg/ai"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	outputFormat string
	outputFile   string
)

var tagCmd = &cobra.Command{
	Use:   "tag [path]",
	Short: "Auto-tag files based on content and metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := filepath.Join(os.Getenv("HOME"), ".smartdir", "smartdir.db")
		if err := storage.InitDB(dbPath); err != nil {
			fmt.Printf("Database initialization failed: %v\n", err)
			return
		}

		path := args[0]

		// Get all files from the database that match the path
		var files []models.File
		result := storage.DB.Where("path LIKE ?", path+"%").Find(&files)
		if result.Error != nil {
			fmt.Printf("Error fetching files: %v\n", result.Error)
			return
		}

		if len(files) == 0 {
			fmt.Println("No files found. Run 'categorize' first.")
			return
		}

		fmt.Printf("Generating tags for %d files...\n", len(files))

		// Process in batches to avoid overwhelming the system
		batchSize := 50
		taggedCount := 0

		for i := 0; i < len(files); i += batchSize {
			end := i + batchSize
			if end > len(files) {
				end = len(files)
			}

			batch := files[i:end]
			for _, file := range batch {
				// Check if file already has tags
				if file.Tags != "" {
					continue
				}

				// Generate tags based on file type and path
				tags := generateTags(file.Path, file.Category)

				// Update the database
				storage.DB.Model(&file).Update("tags", strings.Join(tags, ","))

				fmt.Printf("- %s → %v\n", filepath.Base(file.Path), tags)
				taggedCount++
			}
		}

		fmt.Printf("Tagged %d files successfully\n", taggedCount)

		// Update Meilisearch index with tags
		if taggedCount > 0 {
			// Initialize Meilisearch client
			client := meilisearch.New(viper.GetString("search.host"),
				meilisearch.WithAPIKey(viper.GetString("search.api_key")),
			)

			index := client.Index("files")

			// Get all tagged files and update them in Meilisearch
			var taggedFiles []models.File
			storage.DB.Where("tags != ?", "").Find(&taggedFiles)

			docs := make([]map[string]interface{}, 0, len(taggedFiles))
			for _, file := range taggedFiles {
				doc := map[string]interface{}{
					"id":       file.ID,
					"path":     file.Path,
					"category": file.Category,
					"hash":     file.Hash,
					"tags":     file.Tags,
				}
				docs = append(docs, doc)
			}

			if len(docs) > 0 {
				task, err := index.UpdateDocuments(docs, "id")
				if err != nil {
					fmt.Printf("Warning: Failed to update search index: %v\n", err)
				} else {
					fmt.Println("Search index updated with tags")

					// Wait for the task to complete
					_, err = client.WaitForTask(task.TaskUID, 30) // 30 seconds timeout
					if err != nil {
						fmt.Printf("Warning: Task failed: %v\n", err)
					}
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format (text, json, csv)")
	tagCmd.Flags().StringVar(&outputFile, "output", "", "Output file path")
}

func generateTags(filePath, category string) []string {
	// Generate basic tags from filename
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Split name by common separators
	basicTags := []string{}
	for _, part := range strings.FieldsFunc(nameWithoutExt, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	}) {
		// Only add if part is meaningful (length > 2)
		if len(part) > 2 {
			basicTags = append(basicTags, strings.ToLower(part))
		}
	}

	// Add category as a tag
	if category != "" && category != "other" {
		basicTags = append(basicTags, category)
	}

	// For certain categories, use AI to generate more meaningful tags
	if category == "images" || category == "documents" {
		aiTags, err := ai.GenerateTags(filePath)
		if err == nil && len(aiTags) > 0 {
			// Combine with basic tags, remove duplicates
			return uniqueTags(append(basicTags, aiTags...))
		}
	}

	return uniqueTags(basicTags)
}

func uniqueTags(tags []string) []string {
	seen := make(map[string]bool)
	unique := []string{}

	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag != "" && !seen[tag] {
			seen[tag] = true
			unique = append(unique, tag)
		}
	}

	return unique
}
