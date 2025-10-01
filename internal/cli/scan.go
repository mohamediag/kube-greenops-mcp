package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"greenops-mcp/internal/krr"
	"gopkg.in/yaml.v3"
)

var (
	namespace     string
	kubeContext   string
	clusterName   string
	strategy      string
	cpuMin        string
	cpuMax        string
	memoryMin     string
	memoryMax     string
	recommendOnly bool
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Execute a KRR scan",
	Long: `Execute a KRR (Kubernetes Resource Recommender) scan to analyze resource usage and get recommendations.

Examples:
  krr-cli scan
  krr-cli scan --namespace default
  krr-cli scan --namespace production --strategy advanced
  krr-cli scan --cpu-min 100m --memory-min 128Mi --recommend-only
  krr-cli scan --output yaml --verbose`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Scan-specific flags
	scanCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace to scan (optional, scans all namespaces if not specified)")
	scanCmd.Flags().StringVar(&kubeContext, "context", "", "Kubernetes context to use (optional, uses current context if not specified)")
	scanCmd.Flags().StringVar(&clusterName, "cluster", "", "Name of the cluster for reporting purposes (optional)")
	scanCmd.Flags().StringVar(&strategy, "strategy", "simple", "Recommendation strategy to use (e.g., 'simple', 'advanced')")
	scanCmd.Flags().StringVar(&cpuMin, "cpu-min", "", "Minimum CPU recommendation threshold (e.g., '100m')")
	scanCmd.Flags().StringVar(&cpuMax, "cpu-max", "", "Maximum CPU recommendation threshold (e.g., '2')")
	scanCmd.Flags().StringVar(&memoryMin, "memory-min", "", "Minimum memory recommendation threshold (e.g., '128Mi')")
	scanCmd.Flags().StringVar(&memoryMax, "memory-max", "", "Maximum memory recommendation threshold (e.g., '4Gi')")
	scanCmd.Flags().BoolVar(&recommendOnly, "recommend-only", false, "Only show resources that have recommendations")
}

func runScan(cmd *cobra.Command, args []string) error {
	// Get executor
	executor, err := getExecutor()
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Build scan options
	options := krr.ScanOptions{
		Namespace:     namespace,
		Context:       kubeContext,
		ClusterName:   clusterName,
		Strategy:      strategy,
		CPUMin:        cpuMin,
		CPUMax:        cpuMax,
		MemoryMin:     memoryMin,
		MemoryMax:     memoryMax,
		Output:        krr.OutputFormat(outputFormat),
		RecommendOnly: recommendOnly,
		Verbose:       verbose,
		NoColor:       true, // Always use no color for CLI output
	}

	if verbose {
		fmt.Printf("Executing KRR scan with options:\n")
		fmt.Printf("  Namespace: %s\n", getStringOrDefault(options.Namespace, "all"))
		fmt.Printf("  Context: %s\n", getStringOrDefault(options.Context, "current"))
		fmt.Printf("  Strategy: %s\n", options.Strategy)
		fmt.Printf("  Output Format: %s\n", string(options.Output))
		if options.CPUMin != "" {
			fmt.Printf("  CPU Min: %s\n", options.CPUMin)
		}
		if options.CPUMax != "" {
			fmt.Printf("  CPU Max: %s\n", options.CPUMax)
		}
		if options.MemoryMin != "" {
			fmt.Printf("  Memory Min: %s\n", options.MemoryMin)
		}
		if options.MemoryMax != "" {
			fmt.Printf("  Memory Max: %s\n", options.MemoryMax)
		}
		fmt.Printf("  Recommend Only: %t\n", options.RecommendOnly)
		fmt.Println()
	}

	// Execute the scan
	result, err := executor.Scan(ctx, options)
	if err != nil {
		return fmt.Errorf("KRR scan failed: %w", err)
	}

	// Output the result
	return outputResult(result)
}

func outputResult(result *krr.ScanResult) error {
	switch outputFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "yaml":
		yamlBytes, err := yaml.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result to YAML: %w", err)
		}
		fmt.Println(string(yamlBytes))
	default:
		// Human-readable output
		fmt.Printf("KRR Scan Results\n")
		fmt.Printf("================\n")
		fmt.Printf("Timestamp: %s\n", result.Timestamp)
		if result.Cluster != "" {
			fmt.Printf("Cluster: %s\n", result.Cluster)
		}
		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Total Resources: %d\n", result.Summary.TotalResources)
		fmt.Printf("  Resources with Recommendations: %d\n", result.Summary.ResourcesWithRecommendations)
		fmt.Printf("  Critical Severity: %d\n", result.Summary.CriticalSeverity)
		fmt.Printf("  High Severity: %d\n", result.Summary.HighSeverity)
		fmt.Printf("  Medium Severity: %d\n", result.Summary.MediumSeverity)
		fmt.Printf("  Low Severity: %d\n", result.Summary.LowSeverity)

		if len(result.Resources) > 0 {
			fmt.Printf("\nResources:\n")
			for i, resource := range result.Resources {
				fmt.Printf("  %d. %s/%s (%s)\n", i+1, resource.Namespace, resource.Name, resource.Kind)
				if resource.Container != "" {
					fmt.Printf("     Container: %s\n", resource.Container)
				}
				if resource.Current.CPU != "" || resource.Current.Memory != "" {
					fmt.Printf("     Current: CPU=%s, Memory=%s\n", 
						getStringOrDefault(resource.Current.CPU, "N/A"),
						getStringOrDefault(resource.Current.Memory, "N/A"))
				}
				if resource.Recommended.CPU != "" || resource.Recommended.Memory != "" {
					fmt.Printf("     Recommended: CPU=%s, Memory=%s\n",
						getStringOrDefault(resource.Recommended.CPU, "N/A"),
						getStringOrDefault(resource.Recommended.Memory, "N/A"))
				}
				if resource.Severity != "" {
					fmt.Printf("     Severity: %s\n", resource.Severity)
				}
				if resource.Reason != "" {
					fmt.Printf("     Reason: %s\n", resource.Reason)
				}
				fmt.Println()
			}
		}
	}

	return nil
}

func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}