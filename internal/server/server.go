package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"greenops-mcp/internal/config"
	"greenops-mcp/internal/krr"

	"github.com/invopop/jsonschema"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func init() {
	jsonschema.Version = "https://json-schema.org/draft-07/schema#"
}

// MCPServer wraps the KRR functionality as an MCP server
type MCPServer struct {
	server   *mcp.Server
	executor krr.Executor
	config   *config.Config
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(cfg *config.Config) (*MCPServer, error) {
	// Create KRR executor
	executor := krr.NewCLIExecutor(cfg.KRRPath, cfg.DefaultTimeout)

	// Create MCP server with stdio transport
	transport := stdio.NewStdioServerTransport()
	server := mcp.NewServer(transport)

	mcpServer := &MCPServer{
		server:   server,
		executor: executor,
		config:   cfg,
	}

	// Register tools
	if err := mcpServer.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return mcpServer, nil
}

// KRRScanArguments defines the arguments for the krr_scan tool
type KRRScanArguments struct {
	Namespace     *string `json:"namespace,omitempty" jsonschema:"description=Kubernetes namespace to scan (optional, scans all namespaces if not specified)"`
	Context       *string `json:"context,omitempty" jsonschema:"description=Kubernetes context to use (optional, uses current context if not specified)"`
	ClusterName   *string `json:"cluster_name,omitempty" jsonschema:"description=Name of the cluster for reporting purposes (optional)"`
	Strategy      *string `json:"strategy,omitempty" jsonschema:"description=Recommendation strategy to use (e.g. 'simple' 'advanced')"`
	CPUMin        *string `json:"cpu_min,omitempty" jsonschema:"description=Minimum CPU recommendation threshold (e.g. '100m')"`
	CPUMax        *string `json:"cpu_max,omitempty" jsonschema:"description=Maximum CPU recommendation threshold (e.g. '2')"`
	MemoryMin     *string `json:"memory_min,omitempty" jsonschema:"description=Minimum memory recommendation threshold (e.g. '128Mi')"`
	MemoryMax     *string `json:"memory_max,omitempty" jsonschema:"description=Maximum memory recommendation threshold (e.g. '4Gi')"`
	OutputFormat  *string `json:"output_format,omitempty" jsonschema:"description=Output format: 'json' or 'yaml' (default: json),enum=json,enum=yaml"`
	RecommendOnly *bool   `json:"recommend_only,omitempty" jsonschema:"description=Only show resources that have recommendations (default: false)"`
	Verbose       *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose output (default: false)"`
	KRRPath       *string `json:"krr_path,omitempty" jsonschema:"description=Override the path to the KRR CLI executable (optional)"`
}

// KRRPathArguments allow overriding the KRR binary location for targeted commands
type KRRPathArguments struct {
	KRRPath *string `json:"krr_path,omitempty" jsonschema:"description=Override the path to the KRR CLI executable (optional)"`
}

// registerTools registers all KRR tools with the MCP server
func (s *MCPServer) registerTools() error {
	// Register krr_scan tool
	if err := s.server.RegisterTool("krr_scan", "Execute a KRR (Kubernetes Resource Recommender) scan to analyze resource usage and get recommendations",
		s.handleScan); err != nil {
		return fmt.Errorf("failed to register krr_scan tool: %w", err)
	}

	// Register krr_validate tool
	if err := s.server.RegisterTool("krr_validate", "Validate that KRR CLI is properly installed and accessible", s.handleValidate); err != nil {
		return fmt.Errorf("failed to register krr_validate tool: %w", err)
	}

	// Register krr_version tool
	if err := s.server.RegisterTool("krr_version", "Get the version of the installed KRR CLI", s.handleVersion); err != nil {
		return fmt.Errorf("failed to register krr_version tool: %w", err)
	}

	// Register krr_strategies tool
	if err := s.server.RegisterTool("krr_strategies", "List available KRR recommendation strategies", s.handleStrategies); err != nil {
		return fmt.Errorf("failed to register krr_strategies tool: %w", err)
	}

	return nil
}

// handleScan handles the krr_scan tool execution
func (s *MCPServer) handleScan(arguments KRRScanArguments) (*mcp.ToolResponse, error) {
	// Create context with default timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.DefaultTimeout)
	defer cancel()

	// Parse arguments into ScanOptions
	options := krr.ScanOptions{
		Output: krr.OutputFormat(s.config.DefaultOutputFormat),
	}

	executor := s.executor
	if arguments.KRRPath != nil && strings.TrimSpace(*arguments.KRRPath) != "" {
		executor = krr.NewCLIExecutor(strings.TrimSpace(*arguments.KRRPath), s.config.DefaultTimeout)
	}

	if arguments.Namespace != nil {
		options.Namespace = *arguments.Namespace
	} else if s.config.DefaultNamespace != "" {
		options.Namespace = s.config.DefaultNamespace
	}

	if arguments.Context != nil {
		options.Context = *arguments.Context
	}

	if arguments.ClusterName != nil {
		options.ClusterName = *arguments.ClusterName
	}

	if arguments.Strategy != nil {
		options.Strategy = *arguments.Strategy
	} else {
		options.Strategy = s.config.DefaultStrategy
	}

	if arguments.CPUMin != nil {
		options.CPUMin = *arguments.CPUMin
	}

	if arguments.CPUMax != nil {
		options.CPUMax = *arguments.CPUMax
	}

	if arguments.MemoryMin != nil {
		options.MemoryMin = *arguments.MemoryMin
	}

	if arguments.MemoryMax != nil {
		options.MemoryMax = *arguments.MemoryMax
	}

	if arguments.OutputFormat != nil {
		options.Output = krr.OutputFormat(*arguments.OutputFormat)
	}

	if arguments.RecommendOnly != nil {
		options.RecommendOnly = *arguments.RecommendOnly
	}

	if arguments.Verbose != nil {
		options.Verbose = *arguments.Verbose
	}

	options.NoColor = s.config.DefaultNoColor

	// Execute the scan
	result, err := executor.Scan(ctx, options)
	if err != nil {
		errorMsg := fmt.Sprintf("KRR scan failed: %v", err)
		if strings.Contains(err.Error(), "executable file not found") {
			errorMsg += "\n\nKRR CLI is not installed or not in PATH. Please install it with:\n  pip install krr\n\nThen verify installation with:\n  krr --version"
		}
		return mcp.NewToolResponse(mcp.NewTextContent(errorMsg)), nil
	}

	// Format the result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Failed to format scan result: %v", err))), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("KRR Scan Results:\n\n%s", string(resultJSON)))), nil
}

// handleValidate handles the krr_validate tool execution
func (s *MCPServer) handleValidate(arguments KRRPathArguments) (*mcp.ToolResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	executor := s.executor
	if arguments.KRRPath != nil && strings.TrimSpace(*arguments.KRRPath) != "" {
		executor = krr.NewCLIExecutor(strings.TrimSpace(*arguments.KRRPath), s.config.DefaultTimeout)
	}

	err := executor.ValidateInstallation(ctx)
	if err != nil {
		errorMsg := fmt.Sprintf("KRR validation failed: %v", err)
		if strings.Contains(err.Error(), "executable file not found") {
			errorMsg += "\n\nKRR CLI is not installed or not in PATH. Please install it with:\n  pip install krr\n\nThen verify installation with:\n  krr --version"
		}
		return mcp.NewToolResponse(mcp.NewTextContent(errorMsg)), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent("KRR CLI is properly installed and accessible")), nil
}

// handleVersion handles the krr_version tool execution
func (s *MCPServer) handleVersion(arguments KRRPathArguments) (*mcp.ToolResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	executor := s.executor
	if arguments.KRRPath != nil && strings.TrimSpace(*arguments.KRRPath) != "" {
		executor = krr.NewCLIExecutor(strings.TrimSpace(*arguments.KRRPath), s.config.DefaultTimeout)
	}

	version, err := executor.GetVersion(ctx)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get KRR version: %v", err)
		if strings.Contains(err.Error(), "executable file not found") {
			errorMsg += "\n\nKRR CLI is not installed or not in PATH. Please install it with:\n  pip install krr\n\nThen verify installation with:\n  krr --version"
		}
		return mcp.NewToolResponse(mcp.NewTextContent(errorMsg)), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("KRR CLI Version: %s", version))), nil
}

// handleStrategies handles the krr_strategies tool execution
func (s *MCPServer) handleStrategies(arguments KRRPathArguments) (*mcp.ToolResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	executor := s.executor
	if arguments.KRRPath != nil && strings.TrimSpace(*arguments.KRRPath) != "" {
		executor = krr.NewCLIExecutor(strings.TrimSpace(*arguments.KRRPath), s.config.DefaultTimeout)
	}

	strategies, err := executor.ListStrategies(ctx)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get KRR strategies: %v", err)
		if strings.Contains(err.Error(), "executable file not found") {
			errorMsg += "\n\nKRR CLI is not installed or not in PATH. Please install it with:\n  pip install krr\n\nThen verify installation with:\n  krr --version"
		}
		return mcp.NewToolResponse(mcp.NewTextContent(errorMsg)), nil
	}

	strategiesJSON, err := json.MarshalIndent(strategies, "", "  ")
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Failed to format strategies: %v", err))), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Available KRR Strategies:\n\n%s", string(strategiesJSON)))), nil
}

// Run starts the MCP server
func (s *MCPServer) Run() error {
	log.Printf("Starting KRR MCP Server %s version %s", s.config.ServerName, s.config.ServerVersion)
	log.Printf("Using KRR CLI at: %s", s.config.KRRPath)

	if err := s.server.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	select {}
}

// Close gracefully shuts down the server
func (s *MCPServer) Close() error {
	// The mcp-golang server doesn't seem to have a Close method in the current API
	// This method is kept for interface compatibility
	return nil
}
