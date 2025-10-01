package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// strategiesCmd represents the strategies command
var strategiesCmd = &cobra.Command{
	Use:   "strategies",
	Short: "List available KRR recommendation strategies",
	Long: `List the available KRR recommendation strategies.

Different strategies may provide different types of recommendations
based on various algorithms and data analysis methods.

Examples:
  krr-cli strategies
  krr-cli strategies --output json
  krr-cli strategies --output yaml
  krr-cli strategies --verbose`,
	RunE: runStrategies,
}

func init() {
	rootCmd.AddCommand(strategiesCmd)
}

func runStrategies(cmd *cobra.Command, args []string) error {
	// Get executor
	executor, err := getExecutor()
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if verbose {
		fmt.Printf("Getting available KRR strategies...\n")
		fmt.Printf("KRR Path: %s\n", krrPath)
		fmt.Println()
	}

	// Get strategies
	strategies, err := executor.ListStrategies(ctx)
	if err != nil {
		return fmt.Errorf("failed to get KRR strategies: %w", err)
	}

	// Output strategies
	switch outputFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(strategies, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal strategies to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "yaml":
		yamlBytes, err := yaml.Marshal(strategies)
		if err != nil {
			return fmt.Errorf("failed to marshal strategies to YAML: %w", err)
		}
		fmt.Println(string(yamlBytes))
	default:
		// Human-readable output
		fmt.Printf("Available KRR Strategies:\n")
		fmt.Printf("========================\n")
		for i, strategy := range strategies {
			fmt.Printf("%d. %s\n", i+1, strategy)
		}
		
		if verbose && len(strategies) > 0 {
			fmt.Println("\nUsage:")
			fmt.Printf("  krr-cli scan --strategy %s\n", strategies[0])
		}
	}

	return nil
}