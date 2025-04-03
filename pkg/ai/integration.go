package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// GroqClient holds the API key, model, and HTTP client
type GroqClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// Structs for Groq request and response
type GroqRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Client instance
var Client *GroqClient

// Init initializes the Groq client using Viper for configuration
func Init() {
	Client = NewGroqClient(
		viper.GetString("groq.api_key"),
		viper.GetString("groq.model"),
	)
}

// NewGroqClient creates a new GroqClient instance
func NewGroqClient(apiKey, model string) *GroqClient {
	return &GroqClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ClassifyFile uses the Groq API to categorize the file
func (g *GroqClient) ClassifyFile(path string) (string, error) {
	filename := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	prompt := fmt.Sprintf(`Analyze the file and categorize it strictly into one of these categories:
[images, documents, videos, audio, code, archives, other].
Consider both filename and extension.
For context:
- images: jpg, png, gif, svg, webp, etc.
- documents: pdf, doc, docx, txt, md, etc.
- videos: mp4, avi, mov, mkv, etc.
- audio: mp3, wav, flac, ogg, etc.
- code: go, py, js, ts, html, css, etc.
- archives: zip, tar, gz, 7z, rar, etc.
- other: any file that doesn't clearly fit the above

Filename: "%s"
Extension: "%s"
Respond ONLY with the single category name, no explanation.`, filename, ext)

	// Groq request body
	requestBody := GroqRequest{
		Model: g.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Correct API endpoint
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var response GroqResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from AI model")
	}

	category := strings.TrimSpace(strings.ToLower(response.Choices[0].Message.Content))
	return validateCategory(category)
}

// validateCategory ensures the category is valid
func validateCategory(category string) (string, error) {
	valid := map[string]bool{
		"images": true, "documents": true, "videos": true,
		"audio": true, "code": true, "archives": true, "other": true,
	}

	if valid[category] {
		return category, nil
	}
	return "other", fmt.Errorf("invalid category: %s", category)
}

// ClassifyFile handles classification with fallback logic
func ClassifyFile(path string) string {
	if Client == nil {
		return fallbackClassification(path)
	}

	category, err := Client.ClassifyFile(path)
	if err != nil {
		fmt.Printf("Error: %v. Using fallback classification.\n", err)
		return fallbackClassification(path)
	}
	return category
}

// Fallback classification based on file extension
func fallbackClassification(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return "images"
	case ".doc", ".docx", ".pdf", ".txt", ".rtf":
		return "documents"
	case ".mp4", ".mov", ".avi", ".mkv", ".webm":
		return "videos"
	case ".mp3", ".wav", ".flac", ".aac":
		return "audio"
	case ".go", ".py", ".js", ".java", ".cpp":
		return "code"
	case ".zip", ".tar", ".gz", ".7z", ".rar":
		return "archives"
	default:
		return "other"
	}
}

// Add this new function to generate tags
func GenerateTags(path string) ([]string, error) {
	if Client == nil {
		return fallbackTags(path), nil
	}

	filename := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	prompt := fmt.Sprintf(`Generate 3-5 relevant tags for the following file:
Filename: "%s"
Extension: "%s"
File type: %s

Respond ONLY with comma-separated tags in lowercase, no explanation.
Example: "business,finance,report,quarterly"`, filename, ext, getCategoryFromExt(ext))

	// Groq request body
	requestBody := GroqRequest{
		Model: Client.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fallbackTags(path), fmt.Errorf("failed to marshal request: %v", err)
	}

	// Correct API endpoint
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fallbackTags(path), fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+Client.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := Client.httpClient.Do(req)
	if err != nil {
		return fallbackTags(path), fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackTags(path), fmt.Errorf("API returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallbackTags(path), fmt.Errorf("failed to read response: %v", err)
	}

	var response GroqResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fallbackTags(path), fmt.Errorf("failed to parse response: %v", err)
	}

	if len(response.Choices) == 0 {
		return fallbackTags(path), fmt.Errorf("no response from AI model")
	}

	tagsResponse := strings.TrimSpace(response.Choices[0].Message.Content)
	tags := strings.Split(tagsResponse, ",")

	// Clean up tags
	cleanTags := []string{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	return cleanTags, nil
}

// Fallback tags based on file extension and name
func fallbackTags(path string) []string {
	ext := strings.ToLower(filepath.Ext(path))
	filename := filepath.Base(path)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Default tags based on extension
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return []string{"image", "photo"}
	case ".doc", ".docx", ".pdf":
		return []string{"document"}
	case ".mp4", ".mov", ".avi":
		return []string{"video"}
	case ".mp3", ".wav":
		return []string{"audio"}
	case ".go", ".py", ".js":
		return []string{"code", "programming"}
	}

	// Add filename components as tags
	nameTags := []string{}
	parts := strings.FieldsFunc(nameWithoutExt, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	})

	for _, part := range parts {
		if len(part) > 2 {
			nameTags = append(nameTags, strings.ToLower(part))
		}
	}

	return nameTags
}

// Helper function to get category from extension
func getCategoryFromExt(ext string) string {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return "images"
	case ".doc", ".docx", ".pdf", ".txt", ".rtf":
		return "documents"
	case ".mp4", ".mov", ".avi", ".mkv", ".webm":
		return "videos"
	case ".mp3", ".wav", ".flac", ".aac":
		return "audio"
	case ".go", ".py", ".js", ".java", ".cpp":
		return "code"
	case ".zip", ".tar", ".gz", ".7z", ".rar":
		return "archives"
	default:
		return "other"
	}
}
