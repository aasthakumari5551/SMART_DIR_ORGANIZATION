package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smazmi/smartdir-proto/models"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	dryRun  bool
	confirm bool
)

var deduplicateCmd = &cobra.Command{
	Use:   "deduplicate [path]",
	Short: "Find and remove duplicate files",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := filepath.Join(os.Getenv("HOME"), ".smartdir", "smartdir.db")
		if err := storage.InitDB(dbPath); err != nil {
			fmt.Printf("Database initialization failed: %v\n", err)
			return
		}

		path := args[0]
		duplicates, totalSize, err := findDuplicates(path)
		if err != nil {
			fmt.Printf("Error finding duplicates: %v\n", err)
			return
		}

		fmt.Printf("Found %d duplicate files (%.2f MB)\n", len(duplicates), float64(totalSize)/(1024*1024))

		if len(duplicates) == 0 {
			return
		}

		if dryRun || !confirm {
			fmt.Println("Run with --confirm to remove duplicates")
			// Print some examples
			count := 0
			for hash, files := range duplicates {
				if count >= 3 {
					break
				}
				fmt.Printf("\nDuplicate set with hash %s:\n", hash[:8])
				for i, file := range files {
					if i == 0 {
						fmt.Printf("  Keep: %s\n", file)
					} else {
						fmt.Printf("  Remove: %s\n", file)
					}
				}
				count++
			}
			if count < len(duplicates) {
				fmt.Printf("\n... and %d more duplicate sets\n", len(duplicates)-count)
			}
			return
		}

		// Perform actual deduplication
		removed := removeDuplicates(duplicates)
		fmt.Printf("Removed %d duplicate files\n", removed)
	},
}

func init() {
	rootCmd.AddCommand(deduplicateCmd)
	deduplicateCmd.Flags().BoolVar(&dryRun, "dry-run", true, "Only show what would be done")
	deduplicateCmd.Flags().BoolVar(&confirm, "confirm", false, "Actually remove duplicate files")
}

func findDuplicates(rootPath string) (map[string][]string, int64, error) {
	var files []models.File
	result := storage.DB.Where("path LIKE ?", rootPath+"%").Find(&files)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("database query failed: %v", result.Error)
	}

	// Group files by hash
	hashMap := make(map[string][]string)
	for _, file := range files {
		hashMap[file.Hash] = append(hashMap[file.Hash], file.Path)
	}

	// Filter out non-duplicates
	duplicates := make(map[string][]string)
	var totalSize int64
	for hash, paths := range hashMap {
		if len(paths) > 1 {
			duplicates[hash] = paths
			// Get size of first file
			info, err := os.Stat(paths[0])
			if err == nil {
				totalSize += info.Size() * int64(len(paths)-1) // Only count extra copies
			}
		}
	}

	return duplicates, totalSize, nil
}

func removeDuplicates(duplicates map[string][]string) int {
	removed := 0
	for _, paths := range duplicates {
		// Keep the first file, remove others
		for i := 1; i < len(paths); i++ {
			if err := os.Remove(paths[i]); err == nil {
				// Remove from database
				storage.DB.Where("path = ?", paths[i]).Delete(&models.File{})
				removed++
			}
		}
	}
	return removed
}
