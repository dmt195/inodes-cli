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

func main() {
	mcpServer := server.NewMCPServer(
		"imagenodes-mcp-server",
		version,
		server.WithToolCapabilities(true),
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
			mcp.Description("The pipeline UUID"),
			mcp.Required(),
		),
	), handleDescribePipeline)

	mcpServer.AddTool(mcp.NewTool("run_pipeline",
		mcp.WithDescription("Execute an image processing pipeline with the given parameters. Returns the result image (as a URL or base64) along with processing time and billing info. Use describe_pipeline first to learn the required parameters."),
		mcp.WithString("pipeline_id",
			mcp.Description("The pipeline UUID"),
			mcp.Required(),
		),
		mcp.WithObject("params",
			mcp.Description("Key-value parameters for the pipeline (from describe_pipeline). Image params should be asset UUIDs (from upload_image)."),
		),
		mcp.WithBoolean("base64",
			mcp.Description("If true, return the image as base64 instead of a URL (default false)"),
		),
	), handleRunPipeline)

	mcpServer.AddTool(mcp.NewTool("upload_image",
		mcp.WithDescription("Upload a local image file as an ephemeral asset (expires in 24h). Returns an asset UUID that can be used as an image parameter in run_pipeline."),
		mcp.WithString("file_path",
			mcp.Description("Absolute path to the image file to upload"),
			mcp.Required(),
		),
	), handleUploadImage)

	if err := server.ServeStdio(mcpServer); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
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

// --- helpers ---

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
