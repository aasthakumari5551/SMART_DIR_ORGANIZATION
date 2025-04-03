package storage

import (
	"fmt"

	"github.com/smazmi/smartdir-proto/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(path string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	if err := DB.AutoMigrate(&models.File{}, &models.FileAccess{}); err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	return nil
}
