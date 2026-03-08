package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <pipeline.json>",
		Short: "Validate a pipeline definition without executing it",
		Args:  cobra.ExactArgs(1),
		RunE:  runValidate,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
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

	asJSON, _ := cmd.Flags().GetBool("json")

	pipeline, err := readPipelineJSON(args[0])
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Validating pipeline... ")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	result, err := c.ValidatePipeline(pipeline)
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return err
	}

	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if asJSON {
		return output.PrintJSON(result)
	}

	output.PrintValidateResult(result)
	return nil
}

// readPipelineJSON reads a pipeline JSON file and wraps it in {"pipeline": ...}
func readPipelineJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	// If the file already has a "pipeline" key, use it as-is
	if _, ok := raw["pipeline"]; ok {
		return raw, nil
	}

	// Otherwise wrap the entire object
	return map[string]any{"pipeline": raw}, nil
}
