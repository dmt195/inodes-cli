package commands

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteEvaluateOutputs(t *testing.T) {
	png := base64.StdEncoding.EncodeToString([]byte("img"))
	dir := t.TempDir()

	t.Run("single output goes to -o", func(t *testing.T) {
		out := filepath.Join(dir, "single.png")
		written, err := writeEvaluateOutputs(map[string]string{"output": png}, out, "")
		if err != nil || len(written) != 1 || written[0] != out {
			t.Fatalf("written %v, err %v", written, err)
		}
	})

	t.Run("every output goes to the output dir by name", func(t *testing.T) {
		outDir := filepath.Join(dir, "panels")
		written, err := writeEvaluateOutputs(map[string]string{"output": png, "output-left": png}, "output.png", outDir)
		if err != nil || len(written) != 2 {
			t.Fatalf("written %v, err %v", written, err)
		}
		for _, name := range []string{"output.png", "output-left.png"} {
			if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
				t.Errorf("missing %s: %v", name, err)
			}
		}
	})

	t.Run("multiple outputs without an output dir is an error", func(t *testing.T) {
		if _, err := writeEvaluateOutputs(map[string]string{"a": png, "b": png}, "output.png", ""); err == nil {
			t.Fatal("expected an error asking for --output-dir")
		}
	})

	t.Run("output names can't escape the output dir", func(t *testing.T) {
		if _, err := writeEvaluateOutputs(map[string]string{"../evil": png, "b": png}, "", dir); err == nil {
			t.Fatal("expected an error for an unsafe output name")
		}
	})
}
