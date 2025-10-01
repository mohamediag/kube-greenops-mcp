package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"greenops-mcp/internal/config"
	"greenops-mcp/internal/krr"
	"greenops-mcp/internal/server"
)

const (
	defaultConfigPath = ""
)

func main() {
	// Define command line flags
	var (
		configPath = flag.String("config", defaultConfigPath, "Path to configuration file (optional)")
		krrPath    = flag.String("krr-path", "", "Path to KRR CLI executable (overrides config)")
		timeout    = flag.Duration("timeout", 0, "Default timeout for KRR operations (overrides config)")
		logLevel   = flag.String("log-level", "", "Log level: debug, info, warn, error (overrides config)")
		validate   = flag.Bool("validate", false, "Validate KRR installation and exit")
		version    = flag.Bool("version", false, "Show version and exit")
		help       = flag.Bool("help", false, "Show help message")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "KRR MCP Server - Expose KRR (Kubernetes Resource Recommender) functionality via MCP protocol\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  KRR_TIMEOUT        Default timeout for KRR operations (e.g., '5m')\n")
		fmt.Fprintf(os.Stderr, "  KRR_STRATEGY       Default recommendation strategy\n")
		fmt.Fprintf(os.Stderr, "  KRR_NAMESPACE      Default namespace to scan\n")
		fmt.Fprintf(os.Stderr, "  KRR_OUTPUT_FORMAT  Default output format (json or yaml)\n")
		fmt.Fprintf(os.Stderr, "  KRR_LOG_LEVEL      Log level (debug, info, warn, error)\n")
		fmt.Fprintf(os.Stderr, "  KRR_LOG_FILE       Log file path\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                                    # Start server with default config\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -config /path/to/config.json      # Start server with custom config\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -timeout 10m                      # Override default timeout\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -validate                         # Validate KRR installation\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -version                          # Show version information\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *version {
		fmt.Printf("KRR MCP Server v1.0.0\n")
		fmt.Printf("Built with mcp-golang\n")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Load configuration from environment variables
	cfg.LoadFromEnvironment()

	// Override configuration with command line flags
	if *krrPath != "" {
		cfg.KRRPath = *krrPath
	}
	if *timeout > 0 {
		cfg.DefaultTimeout = *timeout
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg)

	// If validate flag is set, just validate and exit
	if *validate {
		if err := validateKRRInstallation(cfg); err != nil {
			log.Fatalf("KRR validation failed: %v", err)
		}
		fmt.Println("KRR CLI is properly installed and accessible")
		os.Exit(0)
	}

	// Create and start MCP server
	if err := runServer(cfg); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// setupLogging configures logging based on the configuration
func setupLogging(cfg *config.Config) {
	// Set log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// If log file is specified, write to file
	if cfg.LogFile != "" {
		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file %s: %v", cfg.LogFile, err)
		} else {
			log.SetOutput(file)
		}
	}

	// Log level filtering would be implemented here if needed
	// For now, we just use the standard log package
}

// validateKRRInstallation validates that KRR CLI is accessible
func validateKRRInstallation(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a temporary KRR executor for validation
	executor := krr.NewCLIExecutor(cfg.KRRPath, cfg.DefaultTimeout)

	// Validate installation
	if err := executor.ValidateInstallation(ctx); err != nil {
		return err
	}

	// Get version for additional validation
	version, err := executor.GetVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get KRR version: %w", err)
	}

	fmt.Printf("KRR CLI Version: %s\n", version)
	fmt.Printf("KRR CLI Path: %s (from PATH)\n", cfg.KRRPath)

	return nil
}

// runServer creates and runs the MCP server
func runServer(cfg *config.Config) error {
	// Create MCP server
	mcpServer, err := server.NewMCPServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Start server in a goroutine
	err = mcpServer.Run()

	return err

}
