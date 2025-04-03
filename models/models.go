package models

type FileDocument struct {
	ID       uint   `json:"id"`
	Path     string `json:"path"`
	Category string `json:"category"`
	Hash     string `json:"hash"`
}
