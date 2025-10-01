package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get KRR CLI version",
	Long: `Get the version of the installed KRR CLI.

This command queries the KRR CLI executable for its version information.

Examples:
  krr-cli version
  krr-cli version --krr-path /usr/local/bin/krr
  krr-cli version --verbose`,
	RunE: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	// Get executor
	executor, err := getExecutor()
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if verbose {
		fmt.Printf("Getting KRR CLI version...\n")
		fmt.Printf("KRR Path: %s\n", krrPath)
		fmt.Println()
	}

	// Get version
	version, err := executor.GetVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get KRR version: %w", err)
	}

	// Output version information
	fmt.Printf("KRR CLI Version: %s\n", version)
	
	if verbose {
		fmt.Printf("KRR CLI Path: %s\n", krrPath)
		fmt.Printf("CLI Tool Version: 1.0.0\n")
	}

	return nil
}