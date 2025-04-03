package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/smazmi/smartdir-proto/internal/core"
	"github.com/smazmi/smartdir-proto/pkg/storage"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor [path]",
	Short: "Monitor a directory for changes and auto-update",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := filepath.Join(os.Getenv("HOME"), ".smartdir", "smartdir.db")
		if err := storage.InitDB(dbPath); err != nil {
			fmt.Printf("Database initialization failed: %v\n", err)
			return
		}

		path := args[0]
		if err := startMonitoring(path); err != nil {
			fmt.Printf("Monitoring error: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}

func startMonitoring(root string) error {
	// Create new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Add all directories to the watcher
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to add directories to watcher: %v", err)
	}

	// Process batches of events periodically rather than immediately
	pendingFiles := make(map[string]time.Time)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Printf("Monitoring %s for changes. Press Ctrl+C to stop.\n", root)

	// Watch for events and errors
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only care about create and modify events for files
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				info, err := os.Stat(event.Name)
				if err == nil && !info.IsDir() {
					pendingFiles[event.Name] = time.Now()
				} else if err == nil && info.IsDir() && event.Op&fsnotify.Create != 0 {
					// Add new directories to the watcher
					watcher.Add(event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("Watcher error: %v\n", err)

		case <-ticker.C:
			// Process any pending files
			if len(pendingFiles) > 0 {
				now := time.Now()
				filesToProcess := []string{}

				// Only process files that haven't been modified for at least 5 seconds
				for file, lastModified := range pendingFiles {
					if now.Sub(lastModified) >= 5*time.Second {
						filesToProcess = append(filesToProcess, file)
						delete(pendingFiles, file)
					}
				}

				if len(filesToProcess) > 0 {
					fmt.Printf("Processing %d changed files...\n", len(filesToProcess))
					for _, file := range filesToProcess {
						// Process each file
						if err := core.CategorizeFile(file); err != nil {
							fmt.Printf("Error processing %s: %v\n", file, err)
						}
					}
					fmt.Println("Finished processing changed files")
				}
			}
		}
	}
}
