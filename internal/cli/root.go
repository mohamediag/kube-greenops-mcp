package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"greenops-mcp/internal/config"
	"greenops-mcp/internal/krr"
)

var (
	configPath  string
	krrPath     string
	timeout     string
	verbose     bool
	outputFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "krr-cli",
	Short: "KRR CLI tool for testing KRR functionality",
	Long: `A CLI tool for testing KRR (Kubernetes Resource Recommender) functionality.
This tool allows you to test KRR operations directly without going through the MCP protocol.

Examples:
  krr-cli scan --namespace default
  krr-cli validate
  krr-cli version
  krr-cli strategies`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (optional)")
	rootCmd.PersistentFlags().StringVar(&krrPath, "krr-path", "krr", "path to KRR CLI executable")
	rootCmd.PersistentFlags().StringVar(&timeout, "timeout", "5m", "timeout for KRR operations")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "output format (json or yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// This function can be expanded to load configuration from files
	// For now, we'll use the flags directly
}

// getExecutor creates and returns a KRR executor based on the current configuration
func getExecutor() (krr.Executor, error) {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with command line flags
	if krrPath != "" && krrPath != "krr" {
		cfg.KRRPath = krrPath
	}

	if timeout != "" && timeout != "5m" {
		if parsedTimeout, err := parseTimeout(timeout); err == nil {
			cfg.DefaultTimeout = parsedTimeout
		}
	}

	// Create executor
	executor := krr.NewCLIExecutor(cfg.KRRPath, cfg.DefaultTimeout)
	return executor, nil
}

// parseTimeout is a simple helper to parse timeout strings
func parseTimeout(timeoutStr string) (time.Duration, error) {
	return time.ParseDuration(timeoutStr)
}