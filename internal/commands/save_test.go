package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// exportedPipeline mirrors the shape written by 'inodes export' (including the
// graph's own id, which the server ignores on update).
const exportedPipeline = `{"pipeline": {"id": "01EXIST", "nodes": {}, "values": {}, "connectionMapFwd": {}, "connectionMapRev": {}}}`

type recordedRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

// runSaveCmd executes 'inodes save' against a fake API server and returns the
// request it received plus captured stdout/stderr.
func runSaveCmd(t *testing.T, handler func(w http.ResponseWriter), args ...string) (*recordedRequest, string, string, error) {
	t.Helper()
	// Keep the developer's real config out of reach (on macOS
	// os.UserConfigDir ignores XDG_CONFIG_HOME and uses $HOME).
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("INODES_API_KEY", "")
	t.Setenv("INODES_BASE_URL", "")

	var got *recordedRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &recordedRequest{Method: r.Method, Path: r.URL.Path}
		json.NewDecoder(r.Body).Decode(&rec.Body)
		got = rec
		w.Header().Set("Content-Type", "application/json")
		handler(w)
	}))
	defer ts.Close()

	file := filepath.Join(t.TempDir(), "pipeline.json")
	if err := os.WriteFile(file, []byte(exportedPipeline), 0644); err != nil {
		t.Fatal(err)
	}

	root := &cobra.Command{Use: "inodes", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("base-url", "", "")
	root.AddCommand(NewSaveCmd())

	var stderr bytes.Buffer
	root.SetErr(&stderr)
	root.SetArgs(append([]string{"save", file, "--api-key", "test-key", "--base-url", ts.URL}, args...))

	// output.Print* write to os.Stdout directly, so capture it.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	runErr := root.Execute()
	w.Close()
	os.Stdout = origStdout
	stdout, _ := io.ReadAll(r)

	return got, string(stdout), stderr.String(), runErr
}

func okUpdate(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"error": false, "message": "Pipeline updated successfully", "data": {"id": "01EXIST", "version": 4, "evaluate_url": "/api/v1/pipelines/01EXIST/evaluate", "describe_url": "/api/v1/pipelines/01EXIST/describe"}}`)
}

func TestSave_CreatePath(t *testing.T) {
	req, stdout, _, err := runSaveCmd(t, func(w http.ResponseWriter) {
		io.WriteString(w, `{"error": false, "message": "ok", "data": {"id": "01NEW", "name": "Fresh", "evaluate_url": "/e", "describe_url": "/d"}}`)
	}, "--name", "Fresh", "--description", "A new one")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != http.MethodPost || req.Path != "/api/v1/pipeline/save" {
		t.Errorf("expected POST /api/v1/pipeline/save, got %s %s", req.Method, req.Path)
	}
	if req.Body["name"] != "Fresh" || req.Body["description"] != "A new one" {
		t.Errorf("unexpected body: %v", req.Body)
	}
	if !strings.Contains(stdout, "Pipeline saved") || !strings.Contains(stdout, "01NEW") {
		t.Errorf("unexpected stdout: %q", stdout)
	}
}

func TestSave_CreateRequiresName(t *testing.T) {
	req, _, _, err := runSaveCmd(t, okUpdate)
	if err == nil || !strings.Contains(err.Error(), `"name"`) || !strings.Contains(err.Error(), "--id") {
		t.Fatalf("expected missing-name error mentioning --id, got %v", err)
	}
	if req != nil {
		t.Errorf("no request should be sent, got %+v", req)
	}
}

func TestSave_ExpectedVersionRequiresID(t *testing.T) {
	_, _, _, err := runSaveCmd(t, okUpdate, "--name", "x", "--expected-version", "2")
	if err == nil || !strings.Contains(err.Error(), "--id") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestSave_UpdatePath(t *testing.T) {
	req, stdout, stderr, err := runSaveCmd(t, okUpdate, "--id", "01EXIST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != http.MethodPut || req.Path != "/api/v1/pipelines/01EXIST" {
		t.Errorf("expected PUT /api/v1/pipelines/01EXIST, got %s %s", req.Method, req.Path)
	}
	p, ok := req.Body["pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("expected unwrapped pipeline object in body, got %v", req.Body)
	}
	if _, nested := p["pipeline"]; nested {
		t.Errorf("exported envelope must be unwrapped, got %v", p)
	}
	if _, ok := p["nodes"]; !ok {
		t.Errorf("expected graph nodes in body, got %v", p)
	}
	if _, ok := req.Body["version"]; ok {
		t.Errorf("version should be omitted by default, got %v", req.Body["version"])
	}
	if _, ok := req.Body["name"]; ok {
		t.Errorf("name must not be sent on update")
	}
	if strings.Contains(stderr, "Warning") {
		t.Errorf("unexpected warning: %q", stderr)
	}
	if !strings.Contains(stderr, "Updating pipeline") {
		t.Errorf("expected progress message on stderr, got %q", stderr)
	}
	for _, want := range []string{"Pipeline updated", "01EXIST", "Version:  4", "/api/v1/pipelines/01EXIST/evaluate"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q: %q", want, stdout)
		}
	}
}

func TestSave_UpdatePath_JSONAndVersion(t *testing.T) {
	req, stdout, _, err := runSaveCmd(t, okUpdate, "--id", "01EXIST", "--expected-version", "3", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Body["version"] != float64(3) {
		t.Errorf("expected version 3 in body, got %v", req.Body["version"])
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout is not JSON: %v (%q)", err, stdout)
	}
	if out["id"] != "01EXIST" || out["version"] != float64(4) {
		t.Errorf("unexpected JSON output: %v", out)
	}
}

func TestSave_UpdateWarnsOnIgnoredMetadata(t *testing.T) {
	_, _, stderr, err := runSaveCmd(t, okUpdate, "--id", "01EXIST", "--name", "Ignored", "--description", "Also ignored")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "Warning") || !strings.Contains(stderr, "ignored") {
		t.Errorf("expected ignored-metadata warning on stderr, got %q", stderr)
	}
}

func TestSave_UpdateErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   []string
	}{
		{"conflict", http.StatusConflict, []string{"409", "Re-export", "inodes export 01EXIST"}},
		{"not found", http.StatusNotFound, []string{"404", "pipeline not found", "01EXIST"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stdout, _, err := runSaveCmd(t, func(w http.ResponseWriter) {
				w.WriteHeader(tt.status)
				io.WriteString(w, `{"error": true, "message": "nope"}`)
			}, "--id", "01EXIST", "--expected-version", "3")
			if err == nil {
				t.Fatal("expected error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q missing %q", err.Error(), want)
				}
			}
			if stdout != "" {
				t.Errorf("expected no stdout on error, got %q", stdout)
			}
		})
	}
}
