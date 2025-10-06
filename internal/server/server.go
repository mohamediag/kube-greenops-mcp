package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"greenops-mcp/internal/config"
	"greenops-mcp/internal/krr"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer wraps the KRR functionality as an MCP server
type MCPServer struct {
	server     *mcp.Server
	executor   krr.Executor
	config     *config.Config
	httpServer *http.Server
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(cfg *config.Config) (*MCPServer, error) {
	// Create KRR executor
	executor := krr.NewCLIExecutor(cfg.KRRPath, cfg.DefaultTimeout)

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.ServerName,
		Version: cfg.ServerVersion,
	}, nil)

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
	Namespace     *string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace to scan (optional, scans all namespaces if not specified)"`
	Context       *string `json:"context,omitempty" jsonschema:"Kubernetes context to use (optional, uses current context if not specified)"`
	ClusterName   *string `json:"cluster_name,omitempty" jsonschema:"Name of the cluster for reporting purposes (optional)"`
	Strategy      *string `json:"strategy,omitempty" jsonschema:"Recommendation strategy to use (e.g. 'simple' 'advanced')"`
	CPUMin        *string `json:"cpu_min,omitempty" jsonschema:"Minimum CPU recommendation threshold (e.g. '100m')"`
	CPUMax        *string `json:"cpu_max,omitempty" jsonschema:"Maximum CPU recommendation threshold (e.g. '2')"`
	MemoryMin     *string `json:"memory_min,omitempty" jsonschema:"Minimum memory recommendation threshold (e.g. '128Mi')"`
	MemoryMax     *string `json:"memory_max,omitempty" jsonschema:"Maximum memory recommendation threshold (e.g. '4Gi')"`
	OutputFormat  *string `json:"output_format,omitempty" jsonschema:"Output format (fixed to 'table' - this parameter is ignored)"`
	RecommendOnly *bool   `json:"recommend_only,omitempty" jsonschema:"Only show resources that have recommendations (default: false)"`
	Verbose       *bool   `json:"verbose,omitempty" jsonschema:"Enable verbose output (default: false)"`
	KRRPath       *string `json:"krr_path,omitempty" jsonschema:"Override the path to the KRR CLI executable (optional)"`
}

// KRRScanOutput defines the output structure for krr_scan tool
type KRRScanOutput struct {
	Result string `json:"result"`
}

// registerTools registers all KRR tools with the MCP server
func (s *MCPServer) registerTools() error {
	// Register krr_scan tool using AddTool with type-safe handler
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "krr_scan",
		Description: "Execute a KRR (Kubernetes Resource Recommender) scan to analyze resource usage and get recommendations",
	}, s.handleScanTyped)

	return nil
}

// ExecuteScan is a public method for testing purposes
func (s *MCPServer) ExecuteScan(arguments KRRScanArguments) (KRRScanOutput, error) {
	req := &mcp.CallToolRequest{}
	_, output, err := s.handleScanTyped(context.Background(), req, arguments)
	return output, err
}

// handleScanTyped handles the krr_scan tool execution with type-safe API
func (s *MCPServer) handleScanTyped(ctx context.Context, req *mcp.CallToolRequest, arguments KRRScanArguments) (*mcp.CallToolResult, KRRScanOutput, error) {
	// Create context with timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.DefaultTimeout)
		defer cancel()
	}

	// Parse arguments into ScanOptions
	options := krr.ScanOptions{
		Output: krr.OutputTable, // Force table format only
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

	// OutputFormat is ignored - always use table format

	if arguments.RecommendOnly != nil {
		options.RecommendOnly = *arguments.RecommendOnly
	}

	options.NoColor = s.config.DefaultNoColor

	// Execute the scan
	result, err := executor.Scan(ctx, options)
	if err != nil {
		errorMsg := fmt.Sprintf("KRR scan failed: %v", err)
		if strings.Contains(err.Error(), "executable file not found") {
			errorMsg += "\n\nKRR CLI is not installed or not in PATH. Please install it with:\n  pip install krr\n\nThen verify installation with:\n  krr --version"
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: errorMsg},
			},
			IsError: true,
		}, KRRScanOutput{}, nil
	}

	// Format the result based on output format
	var outputText string
	// For table and yaml formats, return raw output directly to save tokens
	if options.Output == krr.OutputTable || options.Output == krr.OutputYAML {
		outputText = fmt.Sprintf("KRR Scan Results:\n\n%s", result.RawOutput)
	} else {
		// For JSON format, return structured data
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to format scan result: %v", err)},
				},
				IsError: true,
			}, KRRScanOutput{}, nil
		}
		outputText = fmt.Sprintf("KRR Scan Results:\n\n%s", string(resultJSON))
	}

	return nil, KRRScanOutput{Result: outputText}, nil
}

// Run starts the MCP server
func (s *MCPServer) Run() error {
	log.Printf("Starting KRR MCP Server %s version %s", s.config.ServerName, s.config.ServerVersion)
	log.Printf("Using KRR CLI at: %s", s.config.KRRPath)

	// Create streamable HTTP handler
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server {
			return s.server
		},
		&mcp.StreamableHTTPOptions{},
	)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", handler.ServeHTTP)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("Server ready to accept MCP requests on http://0.0.0.0:8080/mcp")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for either signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down gracefully", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	case err := <-errChan:
		return err
	}
}

// Close gracefully shuts down the server
func (s *MCPServer) Close() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
