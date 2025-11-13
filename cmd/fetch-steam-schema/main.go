package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"achievo/internal/config"
	"achievo/internal/steam"
	"achievo/internal/storage"
)

func main() {
	appID := flag.String("appid", "", "Steam App ID to fetch schema for")
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	if *appID == "" {
		fmt.Fprintf(os.Stderr, "Error: -appid is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize storage
	stor, err := storage.New(cfg.MongoDB.URI, cfg.MongoDB.Database, cfg.MongoDB.Timeout)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer stor.Close()

	// Initialize Steam client
	steamClient := steam.New(cfg.Steam.APIKey, stor)

	// Fetch and store schema
	fmt.Printf("Fetching Steam schema for App ID: %s\n", *appID)
	if err := steamClient.FetchAndStoreSchema(*appID); err != nil {
		log.Fatalf("Failed to fetch and store schema: %v", err)
	}

	fmt.Printf("Successfully fetched and stored schema for App ID: %s\n", *appID)
}

