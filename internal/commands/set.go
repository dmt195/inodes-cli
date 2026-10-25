package commands

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <pipeline-id> <node>.<param>=<value>...",
		Short: "Change parameter values in a saved pipeline",
		Long: `Change one or more parameter values in a saved pipeline without exporting
and re-saving it.

<node> is a node ID or display name (case-insensitive; a name must be unique).
<param> may omit the "params_" prefix. The value keeps the parameter's
existing type (number, text or true/false). The pipeline is updated in place
as a new version; it fails with a version conflict if someone else saved it
in the meantime.

Examples:
  inodes set 01KM2XGX2RPYRQ9F7V2ZP3F5TQ leftCrop.width=1320
  inodes set 01KM2XGX2RPYRQ9F7V2ZP3F5TQ "Caption.font=NotoSansJP" caption.shrink_to_fit=true
  inodes set 01KM2XGX2RPYRQ9F7V2ZP3F5TQ leftCrop.start_x=1320 --dry-run`,
		Args:              cobra.MinimumNArgs(2),
		RunE:              runSet,
		ValidArgsFunction: completePipelineIDs,
	}
	cmd.Flags().Bool("dry-run", false, "Show the changes without saving them")
	return cmd
}

func runSet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(
		cmd.Root().PersistentFlags().Lookup("api-key").Value.String(),
		cmd.Root().PersistentFlags().Lookup("base-url").Value.String(),
	)
	if err != nil {
		return err
	}
	if err := cfg.RequireAPIKey(); err != nil {
		return err
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	pipelineID := args[0]
	pipeline, err := c.GetPipeline(pipelineID)
	if err != nil {
		return err
	}
	if pipeline.PipelineData == nil {
		return fmt.Errorf("pipeline has no graph data")
	}

	changes, err := applySet(pipeline.PipelineData, args[1:])
	if err != nil {
		return err
	}
	for _, ch := range changes {
		fmt.Fprintf(os.Stderr, "  %s: %s %s %s\n", ch.Key, ch.Old, tui.SymbolArrow, ch.New)
	}
	if dryRun {
		fmt.Fprintln(os.Stderr, "Dry run: nothing saved.")
		return nil
	}

	result, err := c.UpdatePipeline(pipelineID, pipeline.PipelineData, pipeline.Version)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s Saved as version %d\n", tui.SymbolCheck, result.Version)
	return nil
}

// setChange records one value changed by applySet.
type setChange struct {
	Key, Old, New string
}

// applySet applies "node.param=value" assignments to a pipeline graph in
// place and returns what changed. Every assignment is checked before any is
// applied, so an error leaves the graph unchanged.
func applySet(graph map[string]any, assignments []string) ([]setChange, error) {
	nodes, _ := graph["nodes"].(map[string]any)
	values, _ := graph["values"].(map[string]any)
	if nodes == nil || values == nil {
		return nil, fmt.Errorf("pipeline graph has no nodes or values")
	}

	type update struct {
		key, field string
		value      any
		change     setChange
	}
	var updates []update
	for _, a := range assignments {
		target, raw, ok := strings.Cut(a, "=")
		nodeRef, param, okDot := strings.Cut(target, ".")
		if !ok || !okDot || nodeRef == "" || param == "" {
			return nil, fmt.Errorf("invalid assignment %q: expected node.param=value", a)
		}
		nodeID, err := findNodeID(nodes, nodeRef)
		if err != nil {
			return nil, err
		}

		key, existing := lookupValue(values, nodeID, param)
		if existing == nil {
			return nil, fmt.Errorf("node %q has no parameter %q; it has: %s", nodeID, param, strings.Join(nodeParams(values, nodeID), ", "))
		}
		field, oldVal, newVal, err := convertLike(existing, raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		updates = append(updates, update{key, field, newVal, setChange{key, formatValue(oldVal), formatValue(newVal)}})
	}

	changes := make([]setChange, 0, len(updates))
	for _, u := range updates {
		values[u.key] = map[string]any{u.field: u.value}
		changes = append(changes, u.change)
	}
	return changes, nil
}

// findNodeID resolves a node ID or a unique, case-insensitive display name.
func findNodeID(nodes map[string]any, ref string) (string, error) {
	if _, ok := nodes[ref]; ok {
		return ref, nil
	}
	var matches []string
	for id, n := range nodes {
		node, _ := n.(map[string]any)
		if name, _ := node["name"].(string); strings.EqualFold(name, ref) {
			matches = append(matches, id)
		}
	}
	sort.Strings(matches)
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("no node with ID or name %q", ref)
	default:
		return "", fmt.Errorf("%d nodes are named %q (%s); use a node ID", len(matches), ref, strings.Join(matches, ", "))
	}
}

// lookupValue finds a node's value by connector name, trying a "params_"
// prefix if the name doesn't have one.
func lookupValue(values map[string]any, nodeID, param string) (string, map[string]any) {
	for _, key := range []string{nodeID + "_" + param, nodeID + "_params_" + param} {
		if v, ok := values[key].(map[string]any); ok {
			return key, v
		}
	}
	return "", nil
}

// nodeParams lists the value connector names stored for a node.
func nodeParams(values map[string]any, nodeID string) []string {
	var out []string
	for key := range values {
		if p, ok := strings.CutPrefix(key, nodeID+"_"); ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// convertLike parses raw into the same kind of value as existing, returning
// the value's field name, its old value and the new one.
func convertLike(existing map[string]any, raw string) (string, any, any, error) {
	switch {
	case existing["num"] != nil:
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return "", nil, nil, fmt.Errorf("expected a number, got %q", raw)
		}
		return "num", existing["num"], n, nil
	case existing["bool"] != nil:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return "", nil, nil, fmt.Errorf("expected true or false, got %q", raw)
		}
		return "bool", existing["bool"], b, nil
	case existing["url"] != nil:
		return "url", existing["url"], raw, nil
	case existing["string"] != nil:
		return "string", existing["string"], raw, nil
	}
	// An empty value (e.g. a zero number that was omitted) can't tell us the
	// type, so guess from the input.
	if n, err := strconv.ParseFloat(raw, 64); err == nil {
		return "num", nil, n, nil
	}
	return "string", nil, raw, nil
}

func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "(unset)"
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		return x
	}
	return fmt.Sprintf("%v", v)
}
