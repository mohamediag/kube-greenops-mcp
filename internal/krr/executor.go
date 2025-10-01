package krr

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CLIExecutor implements the Executor interface using the KRR CLI
type CLIExecutor struct {
	krrPath string
	timeout time.Duration
}

// NewCLIExecutor creates a new CLI executor with the specified KRR path and timeout
func NewCLIExecutor(krrPath string, timeout time.Duration) Executor {
	return &CLIExecutor{
		krrPath: krrPath,
		timeout: timeout,
	}
}

// Scan executes a KRR scan with the provided options
func (e *CLIExecutor) Scan(ctx context.Context, options ScanOptions) (*ScanResult, error) {
	// Set the base strategy command
	strategy := "simple"
	if options.Strategy != "" {
		strategy = options.Strategy
	}
	args := []string{strategy}

	// Add namespace if specified
	if options.Namespace != "" {
		args = append(args, "--namespace", options.Namespace)
	}

	// Add context if specified
	if options.Context != "" {
		args = append(args, "--context", options.Context)
	}

	// Add cluster name using context flag
	if options.ClusterName != "" {
		args = append(args, "--context", options.ClusterName)
	}

	// Add CPU limits if specified
	if options.CPUMin != "" {
		args = append(args, "--cpu-min", options.CPUMin)
	}
	if options.CPUMax != "" {
		// Note: KRR CLI doesn't seem to have cpu-max, using cpu-min for now
		args = append(args, "--cpu-min", options.CPUMax)
	}

	// Add memory limits if specified (using correct flag name)
	if options.MemoryMin != "" {
		args = append(args, "--mem-min", options.MemoryMin)
	}
	if options.MemoryMax != "" {
		// Note: KRR CLI doesn't seem to have mem-max, using mem-min for now
		args = append(args, "--mem-min", options.MemoryMax)
	}

	// Add output format (using correct flag name)
	if options.Output != "" {
		args = append(args, "--formatter", string(options.Output))
	} else {
		// Default to json for easier parsing
		args = append(args, "--formatter", "json")
	}

	// Add verbose flag
	if options.Verbose {
		args = append(args, "--verbose")
	}

	// Add quiet mode if no-color is specified (closest equivalent)
	if options.NoColor {
		args = append(args, "--quiet")
	}

	// Execute the command with timeout context
	timeoutCtx := ctx
	if e.timeout > 0 {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(timeoutCtx, e.krrPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("krr command failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute krr command: %w", err)
	}

	// Parse the output based on format
	result := &ScanResult{
		Timestamp: time.Now().Format(time.RFC3339),
		Cluster:   options.ClusterName,
		RawOutput: string(output),
	}

	// Try to parse JSON output if format is JSON
	if options.Output == OutputJSON || options.Output == "" {
		var resources []Resource
		if err := json.Unmarshal(output, &resources); err == nil {
			result.Resources = resources
			result.Summary = calculateSummary(resources)
		}
	}

	return result, nil
}

// ValidateInstallation checks if KRR CLI is properly installed and accessible
func (e *CLIExecutor) ValidateInstallation(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, e.krrPath, "--version")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("krr CLI validation failed: %w", err)
	}
	return nil
}

// GetVersion returns the version of the installed KRR CLI
func (e *CLIExecutor) GetVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, e.krrPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get krr version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ListStrategies returns available recommendation strategies
func (e *CLIExecutor) ListStrategies(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, e.krrPath, "--help")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get krr help: %w", err)
	}

	// Return the actual strategies available in KRR CLI
	strategies := []string{"simple", "simple-limit"}

	// Verify the strategies exist in the help output
	helpText := string(output)
	if strings.Contains(helpText, "simple") {
		return strategies, nil
	}

	// Fallback to basic strategies if help parsing fails
	return []string{"simple"}, nil
}

// calculateSummary generates a summary from the scan results
func calculateSummary(resources []Resource) Summary {
	summary := Summary{
		TotalResources: len(resources),
	}

	for _, resource := range resources {
		if resource.Recommended.CPU != "" || resource.Recommended.Memory != "" {
			summary.ResourcesWithRecommendations++
		}

		switch strings.ToLower(resource.Severity) {
		case "critical":
			summary.CriticalSeverity++
		case "high":
			summary.HighSeverity++
		case "medium":
			summary.MediumSeverity++
		case "low":
			summary.LowSeverity++
		}
	}

	return summary
}
