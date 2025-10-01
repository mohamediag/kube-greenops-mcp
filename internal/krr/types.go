package krr

import (
	"context"
)

// OutputFormat defines the format for KRR CLI output
type OutputFormat string

const (
	OutputJSON OutputFormat = "json"
	OutputYAML OutputFormat = "yaml"
)

// ScanOptions represents options for KRR scanning
type ScanOptions struct {
	Namespace     string       `json:"namespace,omitempty"`
	Output        OutputFormat `json:"output,omitempty"`
	Context       string       `json:"context,omitempty"`
	ClusterName   string       `json:"cluster_name,omitempty"`
	Strategy      string       `json:"strategy,omitempty"`
	CPUMin        string       `json:"cpu_min,omitempty"`
	CPUMax        string       `json:"cpu_max,omitempty"`
	MemoryMin     string       `json:"memory_min,omitempty"`
	MemoryMax     string       `json:"memory_max,omitempty"`
	RecommendOnly bool         `json:"recommend_only,omitempty"`
	Verbose       bool         `json:"verbose,omitempty"`
	NoColor       bool         `json:"no_color,omitempty"`
}

// Resource represents a Kubernetes resource with recommendations
type Resource struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Kind      string                 `json:"kind"`
	Container string                 `json:"container,omitempty"`
	Current   ResourceRequirements   `json:"current"`
	Recommended ResourceRequirements `json:"recommended"`
	Severity  string                 `json:"severity"`
	Reason    string                 `json:"reason"`
}

// ResourceRequirements represents CPU and memory requirements
type ResourceRequirements struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// ScanResult represents the result of a KRR scan
type ScanResult struct {
	Timestamp   string     `json:"timestamp"`
	Cluster     string     `json:"cluster"`
	Resources   []Resource `json:"resources"`
	Summary     Summary    `json:"summary"`
	RawOutput   string     `json:"raw_output,omitempty"`
}

// Summary provides an overview of the scan results
type Summary struct {
	TotalResources       int `json:"total_resources"`
	ResourcesWithRecommendations int `json:"resources_with_recommendations"`
	CriticalSeverity     int `json:"critical_severity"`
	HighSeverity         int `json:"high_severity"`
	MediumSeverity       int `json:"medium_severity"`
	LowSeverity          int `json:"low_severity"`
}

// Executor defines the interface for executing KRR CLI commands
type Executor interface {
	// Scan executes a KRR scan with the provided options
	Scan(ctx context.Context, options ScanOptions) (*ScanResult, error)
	
	// ValidateInstallation checks if KRR CLI is properly installed and accessible
	ValidateInstallation(ctx context.Context) error
	
	// GetVersion returns the version of the installed KRR CLI
	GetVersion(ctx context.Context) (string, error)
	
	// ListStrategies returns available recommendation strategies
	ListStrategies(ctx context.Context) ([]string, error)
}