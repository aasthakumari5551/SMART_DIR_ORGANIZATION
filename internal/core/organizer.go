package core

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"runtime"

	"github.com/meilisearch/meilisearch-go"
	"github.com/smazmi/smartdir-proto/models"
	"github.com/smazmi/smartdir-proto/pkg/ai"
	"github.com/smazmi/smartdir-proto/pkg/hashing"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/viper"
)

// Use a worker pool for parallel processing
func CategorizeFiles(root string) error {
	// Initialize Meilisearch client
	client := meilisearch.New(viper.GetString("search.host"),
		meilisearch.WithAPIKey(viper.GetString("search.api_key")),
	)

	// Create or get the files index
	index := client.Index("files")
	_, err := index.GetStats()
	if err != nil {
		// Create index if it doesn't exist
		_, createErr := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        "files",
			PrimaryKey: "id",
		})
		if createErr != nil {
			return fmt.Errorf("failed to create search index: %v", createErr)
		}

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
	}

	filesChan := make(chan string, 100)
	errChan := make(chan error, 10)
	doneChan := make(chan bool)

	// Start worker pool
	workerCount := runtime.NumCPU()
	for range workerCount {
		go func() {
			for path := range filesChan {
				if err := processFile(path, index, client); err != nil {
					errChan <- err
				}
			}
			doneChan <- true
		}()
	}

	// Walk directory and send files to workers
	go func() {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				filesChan <- path
			}
			return nil
		})
		if err != nil {
			errChan <- err
		}
		close(filesChan)
	}()

	// Wait for workers to finish
	for range workerCount {
		<-doneChan
	}

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func CategorizeFile(path string) error {
	// Initialize Meilisearch client
	client := meilisearch.New(viper.GetString("search.host"),
		meilisearch.WithAPIKey(viper.GetString("search.api_key")),
	)

	// Get the files index
	index := client.Index("files")

	// Process the single file
	return processFile(path, index, client)
}

// detectMimeType returns the MIME type of a file based on its extension
func detectMimeType(path string) string {
	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream" // Default MIME type for unknown files
	}
	return mimeType
}

func processFile(path string, index meilisearch.IndexManager, client meilisearch.ServiceManager) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %v", path, err)
	}

	// Calculate file hash
	hash, err := hashing.CalculateHash(path)
	if err != nil {
		return fmt.Errorf("hashing failed for %s: %v", path, err)
	}

	// Classify file using AI
	category := ai.ClassifyFile(path)

	// Add transaction support for database operations
	tx := storage.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	file := models.File{
		Path:     path,
		Category: category,
		Hash:     hash,
		Size:     info.Size(),
		MimeType: detectMimeType(path),
	}

	if result := tx.Create(&file); result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save file %s: %v", path, result.Error)
	}

	// Prepare document for Meilisearch
	doc := map[string]any{
		"id":       file.ID, // Use the auto-incrementing ID from database
		"path":     file.Path,
		"category": file.Category,
		"hash":     file.Hash,
		"tags":     file.Tags, // Include tags field (will be empty initially)
	}

	// Add document to search index with explicit primary key
	task, err := index.AddDocuments([]map[string]any{doc}, "id")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to index file %s: %v", path, err)
	}

	// Wait for indexing to complete (optional but recommended)
	_, err = client.WaitForTask(task.TaskUID, 30) // 30 seconds timeout
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("indexing task failed for %s: %v", path, err)
	}

	return tx.Commit().Error
}
