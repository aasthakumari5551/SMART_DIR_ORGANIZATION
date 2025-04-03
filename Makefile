# Project metadata
APP_NAME := smartdir-proto 
BUILD_DIR := bin
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")

# Docker Compose targets
.PHONY: up down restart logs

up:
	@docker-compose up -d

down:
	@docker-compose down

restart:
	@docker-compose down && docker-compose up -d

logs:
	@docker-compose logs -f

# Go targets
.PHONY: build run clean test

build: $(GO_FILES)
	@echo "Building the Go binary..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) ./main.go

run: build
	@echo "Running the app..."
	@./$(BUILD_DIR)/$(APP_NAME)

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# Linting and formatting
.PHONY: fmt vet tidy

fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

vet:
	@echo "Running Go vet..."
	@go vet ./...

tidy:
	@echo "Tidying Go modules..."
	@go mod tidy

# Helper targets
.PHONY: install-deps 

install-deps:
	@echo "Installing Go dependencies..."
	@go mod download

# All-in-one build and run
all: tidy build run

# Help target
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make up            Start Docker containers"
	@echo "  make down          Stop Docker containers"
	@echo "  make restart       Restart Docker containers"
	@echo "  make logs          Show Docker logs"
	@echo "  make build         Build the Go binary"
	@echo "  make run           Run the Go app"
	@echo "  make clean         Remove build artifacts"
	@echo "  make fmt           Format Go code"
	@echo "  make vet           Lint Go code"
	@echo "  make tidy          Tidy Go modules"
	@echo "  make install-deps  Install Go dependencies"
	@echo "  make all           Build and run the app"
