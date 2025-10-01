package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate KRR CLI installation",
	Long: `Validate that the KRR CLI is properly installed and accessible.

This command checks if:
- KRR CLI executable is found in PATH or specified location
- KRR CLI is executable
- KRR CLI responds to version queries

Examples:
  krr-cli validate
  krr-cli validate --krr-path /usr/local/bin/krr
  krr-cli validate --verbose`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Get executor
	executor, err := getExecutor()
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if verbose {
		fmt.Printf("Validating KRR CLI installation...\n")
		fmt.Printf("KRR Path: %s\n", krrPath)
		fmt.Println()
	}

	// Validate installation
	err = executor.ValidateInstallation(ctx)
	if err != nil {
		fmt.Printf("❌ KRR validation failed: %v\n", err)
		
		// Provide helpful error messages
		if verbose {
			fmt.Println("\nTroubleshooting:")
			fmt.Println("1. Install KRR CLI: pip install krr")
			fmt.Println("2. Verify installation: krr --version") 
			fmt.Println("3. Check PATH: which krr")
			fmt.Println("4. If using virtual environment, ensure it's activated")
		}
		
		return fmt.Errorf("KRR CLI validation failed")
	}

	// Get version for additional validation
	version, err := executor.GetVersion(ctx)
	if err != nil {
		fmt.Printf("⚠️  KRR CLI found but failed to get version: %v\n", err)
		return fmt.Errorf("failed to get KRR version")
	}

	// Success!
	fmt.Printf("✅ KRR CLI is properly installed and accessible\n")
	fmt.Printf("   Version: %s\n", version)
	fmt.Printf("   Path: %s\n", krrPath)

	if verbose {
		fmt.Println("\nNext steps:")
		fmt.Println("- Test a scan: krr-cli scan --help")
		fmt.Println("- List strategies: krr-cli strategies") 
		fmt.Println("- Run MCP server: ./krr-mcp-server")
	}

	return nil
}