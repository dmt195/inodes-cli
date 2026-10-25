package commands

import (
	"testing"

	"github.com/dmt195/inodes-cli/internal/client"
)

func TestFindNodeSchema(t *testing.T) {
	nodes := []client.NodeSchema{{Type: "NodeType.paragraph"}, {Type: "NodeType.text"}}
	for _, name := range []string{"NodeType.paragraph", "paragraph", "Paragraph"} {
		n, err := findNodeSchema(nodes, name)
		if err != nil || n.Type != "NodeType.paragraph" {
			t.Errorf("findNodeSchema(%q) = %v, %v", name, n, err)
		}
	}
	if _, err := findNodeSchema(nodes, "paragraf"); err == nil {
		t.Error("expected an error for an unknown node type")
	}
}
