package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// helper to create a test server that returns JSON wrapped in the standard response
func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func writeJSONResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error":   false,
		"message": "success",
		"data":    data,
	})
}

func TestTestAuth_Success(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/test" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("missing API key header")
		}
		writeJSONResponse(w, http.StatusOK, nil)
	})
	defer ts.Close()

	c := New(ts.URL, "test-key")
	err := c.TestAuth()
	if err != nil {
		t.Fatalf("TestAuth failed: %v", err)
	}
}

func TestTestAuth_Unauthorized(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"error": true, "message": "invalid key"})
	})
	defer ts.Close()

	c := New(ts.URL, "bad-key")
	err := c.TestAuth()
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestListPipelines(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/pipelines" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		offset := r.URL.Query().Get("offset")
		pageSize := r.URL.Query().Get("pageSize")
		if offset != "0" || pageSize != "10" {
			t.Errorf("unexpected query params: offset=%s pageSize=%s", offset, pageSize)
		}

		writeJSONResponse(w, http.StatusOK, map[string]any{
			"pipelines": []map[string]any{
				{"id": "abc-123", "name": "Test Pipeline"},
			},
			"meta": map[string]any{
				"count":       1,
				"offset":      0,
				"pageSize":    10,
				"currentPage": 1,
				"totalPages":  1,
			},
		})
	})
	defer ts.Close()

	c := New(ts.URL, "test-key")
	result, err := c.ListPipelines(0, 10)
	if err != nil {
		t.Fatalf("ListPipelines failed: %v", err)
	}
	if len(result.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(result.Pipelines))
	}
	if result.Pipelines[0].Name != "Test Pipeline" {
		t.Errorf("expected name 'Test Pipeline', got %q", result.Pipelines[0].Name)
	}
}

func TestDescribePipeline(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/pipelines/test-id/describe" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		writeJSONResponse(w, http.StatusOK, map[string]any{
			"id":          "test-id",
			"name":        "My Pipeline",
			"description": "A test pipeline",
			"api_nodes": []map[string]any{
				{"key": "title", "data_type": "string", "default": "Hello"},
				{"key": "size", "data_type": "number", "default": 100},
			},
			"api_image_nodes": []map[string]any{
				{"key": "logo", "required": false},
			},
		})
	})
	defer ts.Close()

	c := New(ts.URL, "test-key")
	desc, err := c.DescribePipeline("test-id")
	if err != nil {
		t.Fatalf("DescribePipeline failed: %v", err)
	}
	if desc.Name != "My Pipeline" {
		t.Errorf("expected name 'My Pipeline', got %q", desc.Name)
	}
	if len(desc.ApiNodes) != 2 {
		t.Fatalf("expected 2 api nodes, got %d", len(desc.ApiNodes))
	}
	if desc.ApiNodes[0].Key != "title" {
		t.Errorf("expected first key 'title', got %q", desc.ApiNodes[0].Key)
	}
	if len(desc.ApiImageNodes) != 1 {
		t.Fatalf("expected 1 image node, got %d", len(desc.ApiImageNodes))
	}
}

func TestEvaluatePipeline(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var params map[string]any
		json.NewDecoder(r.Body).Decode(&params)
		if params["title"] != "Test" {
			t.Errorf("expected title 'Test', got %v", params["title"])
		}

		// Evaluate returns report directly (not wrapped)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"image_details": map[string]any{
				"image_url": "/api/v1/assets/ephemeral/result-123/file",
				"width":     800,
				"height":    600,
				"format":    "png",
			},
			"total_processing_time":  350000000,
			"total_processing_units": 3,
		})
	})
	defer ts.Close()

	c := New(ts.URL, "test-key")
	report, err := c.EvaluatePipeline("test-id", map[string]any{"title": "Test"}, false)
	if err != nil {
		t.Fatalf("EvaluatePipeline failed: %v", err)
	}
	if !report.Success {
		t.Error("expected success=true")
	}
	if report.ImageDetails.Width != 800 {
		t.Errorf("expected width 800, got %d", report.ImageDetails.Width)
	}
	if report.ImageDetails.ImageUrl == "" {
		t.Error("expected non-empty image URL")
	}
}

func TestDownloadFile(t *testing.T) {
	content := []byte("fake image data")
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(content)
	})
	defer ts.Close()

	c := New(ts.URL, "test-key")
	data, ct, err := c.DownloadFile("/some/path")
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("unexpected content")
	}
	if ct != "image/png" {
		t.Errorf("expected content-type image/png, got %s", ct)
	}
}

func TestUploadEphemeral(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/assets/ephemeral" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Fatalf("parsing multipart: %v", err)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("getting form file: %v", err)
		}
		file.Close()

		writeJSONResponse(w, http.StatusCreated, map[string]any{
			"id":         "ephemeral-abc-123",
			"expires_at": "2025-01-02T00:00:00Z",
		})
	})
	defer ts.Close()

	// Create a temp file to upload
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")
	os.WriteFile(tmpFile, []byte("fake png"), 0644)

	c := New(ts.URL, "test-key")
	result, err := c.UploadEphemeral(tmpFile)
	if err != nil {
		t.Fatalf("UploadEphemeral failed: %v", err)
	}
	if result.ID != "ephemeral-abc-123" {
		t.Errorf("expected ID 'ephemeral-abc-123', got %q", result.ID)
	}
}

func TestResolveURL(t *testing.T) {
	c := New("https://imagenodes.com", "test-key")

	// Relative path should get baseURL prepended
	got := c.ResolveURL("/api/v1/assets/ephemeral/abc/file")
	want := "https://imagenodes.com/api/v1/assets/ephemeral/abc/file"
	if got != want {
		t.Errorf("ResolveURL relative: got %q, want %q", got, want)
	}

	// Absolute HTTPS URL should be returned as-is
	absURL := "https://storage.googleapis.com/bucket/file.png"
	got = c.ResolveURL(absURL)
	if got != absURL {
		t.Errorf("ResolveURL absolute https: got %q, want %q", got, absURL)
	}

	// Absolute HTTP URL should be returned as-is
	httpURL := "http://localhost:8081/static/artifacts/test.png"
	got = c.ResolveURL(httpURL)
	if got != httpURL {
		t.Errorf("ResolveURL absolute http: got %q, want %q", got, httpURL)
	}
}

func TestDownloadFile_AbsoluteURL(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("image-data"))
	})
	defer ts.Close()

	c := New("https://should-not-be-used.com", "test-key")
	// Pass the test server's absolute URL directly
	data, ct, err := c.DownloadFile(ts.URL + "/some/file.png")
	if err != nil {
		t.Fatalf("DownloadFile with absolute URL failed: %v", err)
	}
	if string(data) != "image-data" {
		t.Errorf("unexpected content: %q", string(data))
	}
	if ct != "image/png" {
		t.Errorf("unexpected content-type: %s", ct)
	}
}

func TestAPIError_NotFound(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": true, "message": "not found"}`)
	})
	defer ts.Close()

	c := New(ts.URL, "test-key")
	_, err := c.DescribePipeline("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}
