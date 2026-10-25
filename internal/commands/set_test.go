package commands

import (
	"strings"
	"testing"
)

func testGraph() map[string]any {
	return map[string]any{
		"nodes": map[string]any{
			"leftCrop": map[string]any{"id": "leftCrop", "name": "Left panel crop", "node_type": "NodeType.crop"},
			"caption":  map[string]any{"id": "caption", "name": "Caption", "node_type": "NodeType.paragraph"},
			"cap2":     map[string]any{"id": "cap2", "name": "Caption", "node_type": "NodeType.paragraph"},
		},
		"values": map[string]any{
			"leftCrop_params_width":        map[string]any{"num": 1080.0},
			"caption_params_font":          map[string]any{"string": "OpenSans"},
			"caption_params_shrink_to_fit": map[string]any{"bool": false},
		},
	}
}

func TestApplySetByIDAndShortParam(t *testing.T) {
	g := testGraph()
	changes, err := applySet(g, []string{"leftCrop.width=1320"})
	if err != nil {
		t.Fatal(err)
	}
	if v := g["values"].(map[string]any)["leftCrop_params_width"].(map[string]any)["num"]; v != 1320.0 {
		t.Errorf("width = %v", v)
	}
	if len(changes) != 1 || changes[0].Old != "1080" || changes[0].New != "1320" {
		t.Errorf("changes %+v", changes)
	}
}

func TestApplySetByNodeName(t *testing.T) {
	g := testGraph()
	if _, err := applySet(g, []string{"left panel crop.params_width=1000"}); err != nil {
		t.Fatal(err)
	}
}

func TestApplySetKeepsValueType(t *testing.T) {
	g := testGraph()
	if _, err := applySet(g, []string{"caption.font=NotoSansJP", "caption.shrink_to_fit=true"}); err != nil {
		t.Fatal(err)
	}
	vals := g["values"].(map[string]any)
	if vals["caption_params_font"].(map[string]any)["string"] != "NotoSansJP" {
		t.Error("font not set as string")
	}
	if vals["caption_params_shrink_to_fit"].(map[string]any)["bool"] != true {
		t.Error("shrink_to_fit not set as bool")
	}
}

func TestApplySetErrors(t *testing.T) {
	cases := map[string]string{
		"leftCrop.width=wide":   "number",
		"nope.width=1":          "no node",
		"Caption.font=OpenSans": "2 nodes",
		"leftCrop.height=1":     "params_width",
		"leftCrop.width":        "node.param=value",
	}
	for arg, want := range cases {
		if _, err := applySet(testGraph(), []string{arg}); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %v, want it to mention %q", arg, err, want)
		}
	}
}

func TestDefaultAssetFilename(t *testing.T) {
	cases := []struct{ id, server, ctype, want string }{
		{"01A", "iPhone frame.png", "image/png", "iPhone frame.png"},
		{"01A", "../../etc/passwd", "image/png", "passwd"},
		{"01A", "", "image/jpeg", "01A.jpeg"},
		{"01A", "", "", "01A.bin"},
	}
	for _, c := range cases {
		if got := defaultAssetFilename(c.id, c.server, c.ctype); got != c.want {
			t.Errorf("defaultAssetFilename(%q, %q, %q) = %q, want %q", c.id, c.server, c.ctype, got, c.want)
		}
	}
}
