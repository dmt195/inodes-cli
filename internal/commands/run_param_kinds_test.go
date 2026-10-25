package commands

import (
	"strings"
	"testing"

	"github.com/dmt195/inodes-cli/internal/client"
)

func TestCheckParamKinds(t *testing.T) {
	desc := &client.PipelineDescription{
		ApiNodes:      []client.ApiValueDescriptor{{Key: "caption"}},
		ApiImageNodes: []client.ApiImageDescriptor{{Key: "screenshot"}},
	}

	err := checkParamKinds(desc, []string{"screenshot"}, nil)
	if err == nil || !strings.Contains(err.Error(), "--image screenshot=") {
		t.Errorf("image key passed with --param: got %v", err)
	}

	err = checkParamKinds(desc, nil, []string{"caption"})
	if err == nil || !strings.Contains(err.Error(), "--param caption=") {
		t.Errorf("value key passed with --image: got %v", err)
	}

	if err := checkParamKinds(desc, []string{"caption"}, []string{"screenshot"}); err != nil {
		t.Errorf("correct flags rejected: %v", err)
	}
}
