package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func callSave(t *testing.T, args map[string]any) (method, path string, body map[string]any, res *mcp.CallToolResult) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"error": false, "message": "ok", "data": {"id": "01EXIST", "version": 2, "evaluate_url": "/e", "describe_url": "/d"}}`)
	}))
	t.Cleanup(ts.Close)
	// Keep the developer's real config out of reach (on macOS
	// os.UserConfigDir ignores XDG_CONFIG_HOME and uses $HOME).
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("INODES_API_KEY", "test-key")
	t.Setenv("INODES_BASE_URL", ts.URL)

	var req mcp.CallToolRequest
	req.Params.Name = "save_pipeline"
	req.Params.Arguments = args
	res, err := handleSavePipeline(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	return
}

func resultText(res *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

func TestSavePipeline_UpdateWithPipelineID(t *testing.T) {
	// Exported (wrapped) graph should be unwrapped before sending.
	method, path, body, res := callSave(t, map[string]any{
		"pipeline_id": "01EXIST",
		"version":     float64(1),
		"pipeline":    map[string]any{"pipeline": map[string]any{"nodes": map[string]any{}}},
	})
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(res))
	}
	if method != http.MethodPut || path != "/api/v1/pipelines/01EXIST" {
		t.Errorf("expected PUT /api/v1/pipelines/01EXIST, got %s %s", method, path)
	}
	p, _ := body["pipeline"].(map[string]any)
	if _, ok := p["nodes"]; !ok {
		t.Errorf("expected unwrapped graph, got %v", body)
	}
	if body["version"] != float64(1) {
		t.Errorf("expected version 1, got %v", body["version"])
	}
	if !strings.Contains(resultText(res), `"version": 2`) {
		t.Errorf("expected new version in result, got %s", resultText(res))
	}
}

func TestSavePipeline_CreateStillRequiresName(t *testing.T) {
	method, _, _, res := callSave(t, map[string]any{
		"pipeline": map[string]any{"nodes": map[string]any{}},
	})
	if !res.IsError || !strings.Contains(resultText(res), "name is required") {
		t.Errorf("expected name-required error, got %s", resultText(res))
	}
	if method != "" {
		t.Errorf("no request should be sent, got %s", method)
	}
}

func TestSavePipeline_Create(t *testing.T) {
	method, path, body, res := callSave(t, map[string]any{
		"name":     "New",
		"pipeline": map[string]any{"nodes": map[string]any{}},
	})
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(res))
	}
	if method != http.MethodPost || path != "/api/v1/pipeline/save" {
		t.Errorf("expected POST /api/v1/pipeline/save, got %s %s", method, path)
	}
	if body["name"] != "New" {
		t.Errorf("unexpected body %v", body)
	}
}
