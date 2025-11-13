package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	MongoDB   MongoDBConfig   `yaml:"mongodb"`
	Steam     SteamConfig     `yaml:"steam"`
	Detection DetectionConfig `yaml:"detection"`
	Logging   LoggingConfig   `yaml:"logging"`
	Paths     PathsConfig     `yaml:"paths"`
}

// MongoDBConfig contains MongoDB connection settings
type MongoDBConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
	Timeout  int    `yaml:"timeout"` // seconds
}

// SteamConfig contains Steam API settings
type SteamConfig struct {
	APIKey string `yaml:"api_key"`
}

// DetectionConfig contains game detection settings
type DetectionConfig struct {
	ScanInterval int `yaml:"scan_interval"` // seconds
	ProcessCheck bool `yaml:"process_check"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"` // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// PathsConfig contains path settings
type PathsConfig struct {
	GameRules string `yaml:"game_rules"` // directory for game rule files
	Cache     string `yaml:"cache"`       // cache directory
}

// Load loads configuration from a YAML file and environment variables
func Load(path string) (*Config, error) {
	var config Config

	// Load from file if it exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err == nil {
			if err := yaml.Unmarshal(data, &config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables (secrets must come from env)
	if mongoURI := os.Getenv("ACHIEVO_MONGODB_URI"); mongoURI != "" {
		config.MongoDB.URI = mongoURI
	}
	if steamKey := os.Getenv("ACHIEVO_STEAM_API_KEY"); steamKey != "" {
		config.Steam.APIKey = steamKey
	}

	// Set defaults
	if config.Paths.GameRules == "" {
		config.Paths.GameRules = "configs/games"
	}
	if config.Paths.Cache == "" {
		config.Paths.Cache = ".cache"
	}
	if config.Detection.ScanInterval == 0 {
		config.Detection.ScanInterval = 5
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.MongoDB.Database == "" {
		config.MongoDB.Database = "achievo"
	}
	if config.MongoDB.Timeout == 0 {
		config.MongoDB.Timeout = 10
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.MongoDB.URI == "" {
		// Check environment variable
		if uri := os.Getenv("ACHIEVO_MONGODB_URI"); uri != "" {
			c.MongoDB.URI = uri
		} else {
			return fmt.Errorf("mongodb.uri is required (set ACHIEVO_MONGODB_URI env var or in config)")
		}
	}
	if c.Steam.APIKey == "" {
		// Check environment variable
		if key := os.Getenv("ACHIEVO_STEAM_API_KEY"); key != "" {
			c.Steam.APIKey = key
		} else {
			return fmt.Errorf("steam.api_key is required (set ACHIEVO_STEAM_API_KEY env var or in config)")
		}
	}
	return nil
}

