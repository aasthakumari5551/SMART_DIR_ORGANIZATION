package models

import (
	"gorm.io/gorm"
)

type File struct {
	gorm.Model
	Path     string
	Category string
	Hash     string
	Tags     string
	Size     int64
	MimeType string
}

type FileAccess struct {
	gorm.Model
	FilePath    string
	AccessCount int
}
