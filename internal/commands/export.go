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

func NewExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <pipeline-id>",
		Short: "Export a pipeline's graph data as JSON",
		Long: `Export the full graph definition (nodes, values, connections) of a pipeline.

The output is formatted as {"pipeline": {...}} so it can be piped directly
into 'inodes save', 'inodes validate', or 'inodes evaluate'.

Examples:
  inodes export 01ABC... --json > my-pipeline.json
  inodes export 01ABC... --json | inodes save --name "Copy" -
  inodes export 01ABC... -o my-pipeline.json`,
		Args:              cobra.ExactArgs(1),
		RunE:              runExport,
		ValidArgsFunction: completePipelineIDs,
	}
	cmd.Flags().Bool("json", false, "Output raw JSON (default when stdout is not a terminal)")
	cmd.Flags().StringP("output", "o", "", "Write JSON to file instead of stdout")
	return cmd
}

func runExport(cmd *cobra.Command, args []string) error {
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
	outputPath, _ := cmd.Flags().GetString("output")

	fmt.Fprintf(os.Stderr, "Fetching pipeline... ")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	pipeline, err := c.GetPipeline(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return err
	}

	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if pipeline.PipelineData == nil {
		return fmt.Errorf("pipeline has no graph data")
	}

	// Wrap as {"pipeline": {...}} for compatibility with save/validate/evaluate
	wrapped := map[string]any{
		"pipeline": pipeline.PipelineData,
	}

	// Write to file if -o specified
	if outputPath != "" {
		data, err := json.MarshalIndent(wrapped, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding pipeline: %w", err)
		}
		if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outputPath, err)
		}
		fmt.Fprintf(os.Stderr, "%s Wrote %s (%s)\n", tui.SymbolCheck, outputPath, pipeline.Name)
		return nil
	}

	// JSON mode: output to stdout (always JSON when piped)
	if asJSON || !output.IsInteractive() {
		return output.PrintJSON(wrapped)
	}

	// Interactive: show summary + hint
	output.PrintPipelineExport(pipeline)
	return nil
}
