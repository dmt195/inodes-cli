package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmt195/inodes-cli/internal/client"
)

func reportWith(outputs map[string]client.OutputDetails) *client.PipelineReport {
	return &client.PipelineReport{Success: true, Outputs: outputs}
}

func TestPlanOutputWrites_SingleOutput_NoFlags(t *testing.T) {
	r := reportWith(map[string]client.OutputDetails{
		"output": {Format: "png", Width: 100, Height: 100},
	})
	plan, err := planOutputWrites(r, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan["output"] != "output.png" {
		t.Errorf("expected default output.png, got %q", plan["output"])
	}
}

func TestPlanOutputWrites_SingleOutput_BarePath(t *testing.T) {
	r := reportWith(map[string]client.OutputDetails{
		"output": {Format: "png"},
	})
	plan, err := planOutputWrites(r, []string{"result.png"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan["output"] != "result.png" {
		t.Errorf("expected result.png, got %q", plan["output"])
	}
}

func TestPlanOutputWrites_MultiOutput_BarePath_Errors(t *testing.T) {
	r := reportWith(map[string]client.OutputDetails{
		"thumbnail": {Format: "jpeg"},
		"banner":    {Format: "png"},
	})
	_, err := planOutputWrites(r, []string{"result.png"}, "")
	if err == nil {
		t.Fatal("expected error for bare -o against multi-output pipeline")
	}
	msg := err.Error()
	for _, want := range []string{"thumbnail", "banner", "--output-dir", "<name>=<path>"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q: %s", want, msg)
		}
	}
}

func TestPlanOutputWrites_MultiOutput_OutputDir(t *testing.T) {
	dir := t.TempDir()
	r := reportWith(map[string]client.OutputDetails{
		"thumbnail": {Format: "jpeg"},
		"banner":    {Format: "png"},
	})
	plan, err := planOutputWrites(r, nil, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan["thumbnail"] != filepath.Join(dir, "thumbnail.jpeg") {
		t.Errorf("unexpected thumbnail path: %q", plan["thumbnail"])
	}
	if plan["banner"] != filepath.Join(dir, "banner.png") {
		t.Errorf("unexpected banner path: %q", plan["banner"])
	}
}

func TestPlanOutputWrites_MultiOutput_OutputDir_CreatesMissing(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "nested", "dist")
	r := reportWith(map[string]client.OutputDetails{
		"a": {Format: "png"},
	})
	if _, err := planOutputWrites(r, nil, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("expected dir created at %s: err=%v", dir, err)
	}
}

func TestPlanOutputWrites_MultiOutput_PerOutputMappings(t *testing.T) {
	r := reportWith(map[string]client.OutputDetails{
		"thumbnail": {Format: "jpeg"},
		"banner":    {Format: "png"},
	})
	plan, err := planOutputWrites(r, []string{"thumbnail=./t.jpg", "banner=./b.png"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan["thumbnail"] != "./t.jpg" || plan["banner"] != "./b.png" {
		t.Errorf("unexpected plan: %+v", plan)
	}
}

func TestPlanOutputWrites_UnknownOutputName_Errors(t *testing.T) {
	r := reportWith(map[string]client.OutputDetails{
		"thumbnail": {Format: "jpeg"},
	})
	_, err := planOutputWrites(r, []string{"og_image=./og.png"}, "")
	if err == nil {
		t.Fatal("expected error for unknown output name")
	}
	if !strings.Contains(err.Error(), "og_image") {
		t.Errorf("error should name the unknown output, got: %s", err)
	}
}

func TestPlanOutputWrites_MixedBareAndMappings_Errors(t *testing.T) {
	r := reportWith(map[string]client.OutputDetails{
		"thumbnail": {Format: "jpeg"},
		"banner":    {Format: "png"},
	})
	_, err := planOutputWrites(r, []string{"thumbnail=./t.jpg", "lone.png"}, "")
	if err == nil {
		t.Fatal("expected error mixing bare path and name=path")
	}
}

func TestPlanOutputWrites_PartialMapping_SkipsOthers(t *testing.T) {
	r := reportWith(map[string]client.OutputDetails{
		"thumbnail": {Format: "jpeg"},
		"banner":    {Format: "png"},
		"og_image":  {Format: "png"},
	})
	plan, err := planOutputWrites(r, []string{"thumbnail=./t.jpg"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan) != 1 {
		t.Errorf("expected only thumbnail, got plan: %+v", plan)
	}
}
