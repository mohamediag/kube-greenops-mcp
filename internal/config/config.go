package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents the configuration for the KRR MCP server
type Config struct {
	// KRR CLI configuration
	KRRPath         string        `json:"krr_path"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	DefaultStrategy string        `json:"default_strategy"`
	
	// Server configuration
	ServerName        string `json:"server_name"`
	ServerVersion     string `json:"server_version"`
	
	// Default scan options
	DefaultNamespace  string `json:"default_namespace"`
	DefaultOutputFormat string `json:"default_output_format"`
	DefaultNoColor    bool   `json:"default_no_color"`
	
	// Logging
	LogLevel string `json:"log_level"`
	LogFile  string `json:"log_file"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		KRRPath:           "krr", // Assumes krr is in PATH
		DefaultTimeout:    5 * time.Minute,
		DefaultStrategy:   "simple",
		ServerName:        "krr-mcp-server",
		ServerVersion:     "1.0.0",
		DefaultNamespace:  "",
		DefaultOutputFormat: "json",
		DefaultNoColor:    true,
		LogLevel:          "info",
		LogFile:           "",
	}
}

// LoadConfig loads configuration from a file or returns default config
func LoadConfig(configPath string) (*Config, error) {
	// If no config path provided, use default config
	if configPath == "" {
		return DefaultConfig(), nil
	}
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse JSON config
	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Validate and set defaults for missing fields
	if config.KRRPath == "" {
		config.KRRPath = "krr"
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 5 * time.Minute
	}
	if config.DefaultStrategy == "" {
		config.DefaultStrategy = "simple"
	}
	if config.ServerName == "" {
		config.ServerName = "krr-mcp-server"
	}
	if config.ServerVersion == "" {
		config.ServerVersion = "1.0.0"
	}
	if config.DefaultOutputFormat == "" {
		config.DefaultOutputFormat = "json"
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	
	return config, nil
}

// SaveConfig saves configuration to a file
func (c *Config) SaveConfig(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.KRRPath == "" {
		return fmt.Errorf("krr_path cannot be empty")
	}
	
	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be positive")
	}
	
	if c.ServerName == "" {
		return fmt.Errorf("server_name cannot be empty")
	}
	
	if c.ServerVersion == "" {
		return fmt.Errorf("server_version cannot be empty")
	}
	
	// Validate output format
	if c.DefaultOutputFormat != "json" && c.DefaultOutputFormat != "yaml" {
		return fmt.Errorf("default_output_format must be 'json' or 'yaml'")
	}
	
	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("log_level must be one of: debug, info, warn, error")
	}
	
	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./krr-mcp-config.json"
	}
	
	return filepath.Join(homeDir, ".config", "krr-mcp", "config.json")
}

// LoadFromEnvironment loads configuration values from environment variables
func (c *Config) LoadFromEnvironment() {
	if krrPath := os.Getenv("KRR_PATH"); krrPath != "" {
		c.KRRPath = krrPath
	}
	
	if timeout := os.Getenv("KRR_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			c.DefaultTimeout = duration
		}
	}
	
	if strategy := os.Getenv("KRR_STRATEGY"); strategy != "" {
		c.DefaultStrategy = strategy
	}
	
	if namespace := os.Getenv("KRR_NAMESPACE"); namespace != "" {
		c.DefaultNamespace = namespace
	}
	
	if outputFormat := os.Getenv("KRR_OUTPUT_FORMAT"); outputFormat != "" {
		c.DefaultOutputFormat = outputFormat
	}
	
	if logLevel := os.Getenv("KRR_LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}
	
	if logFile := os.Getenv("KRR_LOG_FILE"); logFile != "" {
		c.LogFile = logFile
	}
}