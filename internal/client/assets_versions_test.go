package client

import (
	"net/http"
	"testing"
)

func TestDownloadAsset(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/assets/01ASSET/file" || r.Header.Get("X-API-Key") != "k" {
			t.Errorf("unexpected request %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", `inline; filename="iPhone frame.png"`)
		w.Write([]byte("png-bytes"))
	})
	defer ts.Close()

	data, contentType, filename, err := New(ts.URL, "k").DownloadAsset("01ASSET")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "png-bytes" || contentType != "image/png" || filename != "iPhone frame.png" {
		t.Errorf("got %q, %q, %q", data, contentType, filename)
	}
}

func TestDownloadAssetNotFound(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":true,"message":"asset not found"}`))
	})
	defer ts.Close()
	if _, _, _, err := New(ts.URL, "k").DownloadAsset("nope"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestListAndRestoreVersions(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/pipelines/P1/versions":
			writeJSONResponse(w, http.StatusOK, map[string]any{"versions": []map[string]any{
				{"version": 4, "saved_by_email": "a@example.com", "created_at": "2026-10-24T16:03:11Z"},
				{"version": 3, "saved_by_email": "a@example.com", "created_at": "2026-10-23T10:44:02Z"},
			}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/pipelines/P1/versions/3/restore":
			writeJSONResponse(w, http.StatusOK, map[string]any{"version": 5})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	defer ts.Close()

	c := New(ts.URL, "k")
	versions, err := c.ListVersions("P1")
	if err != nil || len(versions) != 2 || versions[0].Version != 4 {
		t.Fatalf("ListVersions = %+v, %v", versions, err)
	}
	newVersion, err := c.RestoreVersion("P1", 3)
	if err != nil || newVersion != 5 {
		t.Fatalf("RestoreVersion = %d, %v", newVersion, err)
	}
}
