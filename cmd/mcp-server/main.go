package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var version = "dev"

const serverInstructions = `Image Nodes MCP Server — image processing via the Image Nodes API.

## Stored Pipelines (all plans)
Use these tools to run pre-built pipelines saved in a user's account:
1. list_pipelines → discover available pipelines
2. describe_pipeline → get the parameter schema (inputs, types, defaults)
3. upload_image → upload a local image file to get an asset ID
4. run_pipeline → execute with parameters, get result image

## Dynamic Pipelines (paid plans)
Build and execute custom pipelines on the fly from JSON:
1. get_node_schema → discover all available node types and their parameters
2. validate_pipeline → check a pipeline definition for errors
3. estimate_pipeline_cost → estimate cost before executing
4. evaluate_pipeline → execute the pipeline and get the result image
5. save_pipeline → save a pipeline definition to the user's account

## Authentication
Requires an API key set via the INODES_API_KEY environment variable
or configured with 'inodes configure'.`

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		return
	}

	mcpServer := server.NewMCPServer(
		"imagenodes-mcp-server",
		version,
		server.WithToolCapabilities(true),
		server.WithInstructions(serverInstructions),
	)

	mcpServer.AddTool(mcp.NewTool("list_pipelines",
		mcp.WithDescription("List available image processing pipelines. Returns pipeline IDs, names, descriptions, and metadata."),
		mcp.WithNumber("offset",
			mcp.Description("Pagination offset (default 0)"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Number of pipelines per page (default 25)"),
		),
	), handleListPipelines)

	mcpServer.AddTool(mcp.NewTool("describe_pipeline",
		mcp.WithDescription("Get the parameter schema for a pipeline. Returns value parameters (with types and defaults) and image parameters (with required flags). Use this before run_pipeline to understand what inputs are needed."),
		mcp.WithString("pipeline_id",
			mcp.Description("The pipeline ID"),
			mcp.Required(),
		),
	), handleDescribePipeline)

	mcpServer.AddTool(mcp.NewTool("run_pipeline",
		mcp.WithDescription("Execute an image processing pipeline with the given parameters. Returns the result image (as a URL or base64) along with processing time and billing info. Use describe_pipeline first to learn the required parameters."),
		mcp.WithString("pipeline_id",
			mcp.Description("The pipeline ID"),
			mcp.Required(),
		),
		mcp.WithObject("params",
			mcp.Description("Key-value parameters for the pipeline (from describe_pipeline). Image params should be asset IDs (from upload_image)."),
		),
		mcp.WithBoolean("base64",
			mcp.Description("If true, return the image as base64 instead of a URL (default false)"),
		),
	), handleRunPipeline)

	mcpServer.AddTool(mcp.NewTool("upload_image",
		mcp.WithDescription("Upload a local image file as an ephemeral asset (expires in 24h). Returns an asset ID that can be used as an image parameter in run_pipeline."),
		mcp.WithString("file_path",
			mcp.Description("Absolute path to the image file to upload"),
			mcp.Required(),
		),
	), handleUploadImage)

	// LLM Integration endpoints (paid plans) — dynamic pipeline creation
	mcpServer.AddTool(mcp.NewTool("get_node_schema",
		mcp.WithDescription("Discover all available image processing node types and their parameters. Use this to understand what nodes can be used when building a custom pipeline definition. Returns node types, their inputs, outputs, and configurable properties."),
	), handleGetNodeSchema)

	mcpServer.AddTool(mcp.NewTool("validate_pipeline",
		mcp.WithDescription("Validate a custom pipeline JSON structure without executing it. Returns whether the pipeline is valid and any errors found. Use this before evaluate_pipeline to check for issues."),
		mcp.WithObject("pipeline",
			mcp.Description("The pipeline definition as a JSON object"),
			mcp.Required(),
		),
	), handleValidatePipeline)

	mcpServer.AddTool(mcp.NewTool("estimate_pipeline_cost",
		mcp.WithDescription("Estimate the cost (in processing units) of executing a custom pipeline without running it."),
		mcp.WithObject("pipeline",
			mcp.Description("The pipeline definition as a JSON object"),
			mcp.Required(),
		),
	), handleEstimatePipelineCost)

	mcpServer.AddTool(mcp.NewTool("save_pipeline",
		mcp.WithDescription("Save a custom pipeline definition to the user's account. Validates the pipeline before saving. Returns the pipeline ID and convenience URLs for evaluating and describing the saved pipeline. Requires a paid subscription."),
		mcp.WithString("name",
			mcp.Description("Name for the saved pipeline"),
			mcp.Required(),
		),
		mcp.WithString("description",
			mcp.Description("Optional description of what the pipeline does"),
		),
		mcp.WithObject("pipeline",
			mcp.Description("The pipeline definition as a JSON object (nodes, values, connectionMapFwd, connectionMapRev)"),
			mcp.Required(),
		),
	), handleSavePipeline)

	mcpServer.AddTool(mcp.NewTool("evaluate_pipeline",
		mcp.WithDescription("Execute a custom pipeline defined as JSON. Unlike run_pipeline (which runs a stored pipeline by ID), this executes an ad-hoc pipeline definition directly. Requires a paid subscription. Use get_node_schema to discover available nodes, and validate_pipeline to check your definition first."),
		mcp.WithObject("pipeline",
			mcp.Description("The pipeline definition as a JSON object"),
			mcp.Required(),
		),
		mcp.WithBoolean("base64",
			mcp.Description("If true, return the image as base64 instead of a URL (default false)"),
		),
	), handleEvaluatePipeline)

	if err := server.ServeStdio(mcpServer); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`Image Nodes MCP Server %s

An MCP (Model Context Protocol) server for the Image Nodes image processing API.
Communicates over stdio using JSON-RPC.

Tools provided:

  Stored Pipelines (all plans):
    list_pipelines         List available image processing pipelines
    describe_pipeline      Get parameter schema for a pipeline
    run_pipeline           Execute a stored pipeline with parameters
    upload_image           Upload a local image as an ephemeral asset

  Dynamic Pipelines (paid plans):
    get_node_schema        Discover available node types and parameters
    validate_pipeline      Validate a pipeline JSON definition
    estimate_pipeline_cost Estimate execution cost
    evaluate_pipeline      Execute a custom pipeline from JSON
    save_pipeline          Save a pipeline definition to your account

Configuration:
  Set INODES_API_KEY as an environment variable, or run 'inodes configure'.
  Optionally set INODES_BASE_URL to override the default API endpoint.

Usage with Claude Desktop:
  {
    "mcpServers": {
      "imagenodes": {
        "command": "%s",
        "env": { "INODES_API_KEY": "your-key" }
      }
    }
  }
`, version, os.Args[0])
}

// newClient creates an API client from config, returning an error if no API key is set.
func newClient() (*client.Client, error) {
	cfg, err := config.Load("", "")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("no API key configured. Set INODES_API_KEY environment variable or run 'inodes configure'")
	}
	return client.New(cfg.BaseURL, cfg.APIKey), nil
}

func handleListPipelines(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	args := request.GetArguments()
	offset := intArg(args, "offset", 0)
	pageSize := intArg(args, "page_size", 25)

	result, err := c.ListPipelines(offset, pageSize)
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(result)
}

func handleDescribePipeline(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	pipelineID, err := request.RequireString("pipeline_id")
	if err != nil {
		return errorResult(err), nil
	}

	result, err := c.DescribePipeline(pipelineID)
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(result)
}

func handleRunPipeline(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	pipelineID, err := request.RequireString("pipeline_id")
	if err != nil {
		return errorResult(err), nil
	}

	args := request.GetArguments()
	params := make(map[string]any)
	if p, ok := args["params"].(map[string]any); ok {
		params = p
	}

	base64Flag := false
	if b, ok := args["base64"].(bool); ok {
		base64Flag = b
	}

	report, err := c.EvaluatePipeline(pipelineID, params, base64Flag)
	if err != nil {
		return errorResult(err), nil
	}

	// If we got base64 image data, return it as an image content block
	if base64Flag && report.ImageDetails.ImageAsBase64 != "" {
		mimeType := "image/png"
		if report.ImageDetails.Format != "" {
			mimeType = "image/" + report.ImageDetails.Format
		}

		summary := fmt.Sprintf("Pipeline executed successfully. %dx%d %s, %d units billed, %.2fs processing time.",
			report.ImageDetails.Width, report.ImageDetails.Height,
			report.ImageDetails.Format, report.TotalUnitsBillable,
			report.TotalProcessingTime.Seconds())

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.ImageContent{
					Type:     "image",
					Data:     report.ImageDetails.ImageAsBase64,
					MIMEType: mimeType,
				},
				mcp.TextContent{
					Type: "text",
					Text: summary,
				},
			},
		}, nil
	}

	return jsonResult(report)
}

func handleUploadImage(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	filePath, err := request.RequireString("file_path")
	if err != nil {
		return errorResult(err), nil
	}

	result, err := c.UploadEphemeral(filePath)
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(result)
}

func handleGetNodeSchema(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	result, err := c.GetSchemaNodes()
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(result)
}

func handleValidatePipeline(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	args := request.GetArguments()
	pipeline, ok := args["pipeline"].(map[string]any)
	if !ok {
		return errorResult(fmt.Errorf("pipeline must be a JSON object")), nil
	}

	result, err := c.ValidatePipeline(wrapPipeline(pipeline))
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(result)
}

func handleEstimatePipelineCost(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	args := request.GetArguments()
	pipeline, ok := args["pipeline"].(map[string]any)
	if !ok {
		return errorResult(fmt.Errorf("pipeline must be a JSON object")), nil
	}

	result, err := c.EstimatePipelineCost(wrapPipeline(pipeline))
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(result)
}

func handleEvaluatePipeline(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	args := request.GetArguments()
	pipeline, ok := args["pipeline"].(map[string]any)
	if !ok {
		return errorResult(fmt.Errorf("pipeline must be a JSON object")), nil
	}

	base64Flag := false
	if b, ok := args["base64"].(bool); ok {
		base64Flag = b
	}

	result, err := c.EvaluatePipelineJSON(wrapPipeline(pipeline), base64Flag)
	if err != nil {
		return errorResult(err), nil
	}

	if !result.Success {
		return errorResult(fmt.Errorf("pipeline evaluation failed: %s", result.Error)), nil
	}

	if base64Flag && result.Output != "" {
		summary := fmt.Sprintf("Pipeline executed successfully. %d credits used.", result.Cost)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.ImageContent{
					Type:     "image",
					Data:     result.Output,
					MIMEType: "image/png",
				},
				mcp.TextContent{
					Type: "text",
					Text: summary,
				},
			},
		}, nil
	}

	return jsonResult(result)
}

func handleSavePipeline(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c, err := newClient()
	if err != nil {
		return errorResult(err), nil
	}

	name, err := request.RequireString("name")
	if err != nil {
		return errorResult(err), nil
	}

	args := request.GetArguments()
	description := ""
	if d, ok := args["description"].(string); ok {
		description = d
	}

	pipeline, ok := args["pipeline"].(map[string]any)
	if !ok {
		return errorResult(fmt.Errorf("pipeline must be a JSON object")), nil
	}

	result, err := c.SavePipeline(name, description, pipeline)
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(result)
}

// --- helpers ---

// wrapPipeline wraps a raw pipeline object in {"pipeline": ...} if needed.
// The API expects this envelope; the MCP tool schema accepts the raw pipeline object.
func wrapPipeline(pipeline map[string]any) map[string]any {
	if _, ok := pipeline["pipeline"]; ok {
		return pipeline
	}
	return map[string]any{"pipeline": pipeline}
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: err.Error()},
		},
		IsError: true,
	}
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errorResult(fmt.Errorf("encoding result: %w", err)), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: string(data)},
		},
	}, nil
}

func intArg(args map[string]any, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case string:
			if i, err := strconv.Atoi(n); err == nil {
				return i
			}
		}
	}
	return defaultVal
}
