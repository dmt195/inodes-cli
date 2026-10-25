package client

import (
	"net/http"
	"testing"
)

// The schema endpoint's parameters (with defaults, ranges and allowed values)
// must survive decoding so --json and the detail view can show them.
func TestGetSchemaNodesKeepsParameters(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		writeJSONResponse(w, http.StatusOK, []map[string]any{{
			"type":         "NodeType.paragraph",
			"display_name": "Paragraph",
			"description":  "Renders multi-line wrapped text within a bounding box.",
			"category":     "draw",
			"inputs":       []any{},
			"outputs":      []map[string]any{{"name": "output", "type": "Image", "required": false, "description": "Rendered paragraph"}},
			"parameters": []map[string]any{
				{"name": "params_font", "type": "text", "default": "OpenSans", "description": "Font name", "enum": []string{"Cairo", "OpenSans"}},
				{"name": "params_font_size", "type": "number", "default": 16, "min": 1, "max": 200, "description": "Font size", "is_pixel_space": true},
			},
			"examples": []string{"Add captions"},
		}})
	})
	defer ts.Close()

	schema, err := New(ts.URL, "k").GetSchemaNodes()
	if err != nil {
		t.Fatal(err)
	}
	n := schema.Nodes[0]
	if n.DisplayName != "Paragraph" || len(n.Parameters) != 2 || len(n.Outputs) != 1 {
		t.Fatalf("decoded %+v", n)
	}
	font := n.Parameters[0]
	if len(font.Enum) != 2 || font.Enum[1] != "OpenSans" || font.Default != "OpenSans" {
		t.Errorf("font param %+v", font)
	}
	size := n.Parameters[1]
	if size.Min == nil || *size.Min != 1 || size.Max == nil || *size.Max != 200 || !size.IsPixelSpace {
		t.Errorf("font size param %+v", size)
	}
}
