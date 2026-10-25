package commands

import (
	"fmt"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <pipeline.json>",
		Short: "Save a pipeline definition to your account (create or update)",
		Long: `Save a pipeline JSON definition to your Image Nodes account.
The pipeline is validated before saving. Requires a paid subscription.

Without --id a new pipeline is created (counts towards your plan's pipeline
limit) and --name is required. With --id the graph of that existing pipeline
is replaced in place; its name and description are left unchanged, so
--name and --description are ignored.

Use "-" as the file name to read the pipeline from stdin.

Pipeline JSON format:
  The pipeline must contain nodes, values, and connection maps.
  Node IDs must be 12-character alphanumeric strings (NanoIDs) with no
  underscores, since connector IDs use the format "{nodeId}_{suffix}".
  The special nodes "start" and "out" are also allowed.
  Output from 'inodes export' can be used directly; any "id" in the graph
  is ignored on update (the --id value wins).

Nested pipelines:
  To reference another pipeline as a node, use "NodeType.nestedpipeline"
  and set the "{nodeId}_node_id" value to the target pipeline's ID (a
  26-character ULID). The server automatically discovers the target
  pipeline's API parameters and creates the required connector values.

Examples:
  inodes save pipeline.json --name "Social card"
  inodes export 01KM2XGX2RPYRQ9F7V2ZP3F5TQ -o card.json
  # ...edit card.json...
  inodes save card.json --id 01KM2XGX2RPYRQ9F7V2ZP3F5TQ
  inodes save card.json --id 01KM2XGX2RPYRQ9F7V2ZP3F5TQ --expected-version 7`,
		Args: cobra.ExactArgs(1),
		RunE: runSave,
	}
	cmd.Flags().String("name", "", "Pipeline name (required when creating; ignored with --id)")
	cmd.Flags().String("description", "", "Pipeline description (ignored with --id)")
	cmd.Flags().String("id", "", "Update the existing pipeline with this ID instead of creating a new one")
	cmd.Flags().Int("expected-version", 0, "With --id: only update if the stored pipeline is at this version (409 otherwise)")
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.RegisterFlagCompletionFunc("id", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completePipelineIDs(cmd, nil, toComplete)
	})
	return cmd
}

func runSave(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	expectedVersion, _ := cmd.Flags().GetInt("expected-version")
	asJSON, _ := cmd.Flags().GetBool("json")
	stderr := cmd.ErrOrStderr()

	if id == "" {
		if name == "" {
			return fmt.Errorf(`required flag "name" not set (or pass --id <pipeline-id> to update an existing pipeline)`)
		}
		if cmd.Flags().Changed("expected-version") {
			return fmt.Errorf("--expected-version can only be used with --id")
		}
	} else {
		if expectedVersion < 0 {
			return fmt.Errorf("--expected-version must be a positive integer")
		}
		if cmd.Flags().Changed("name") || cmd.Flags().Changed("description") {
			fmt.Fprintln(stderr, tui.Warning.Render("Warning: --name and --description are ignored when updating with --id (only the graph is replaced)"))
		}
	}

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

	pipeline, err := readPipelineJSON(args[0])
	if err != nil {
		return err
	}

	// Extract the inner pipeline object if wrapped
	inner, ok := pipeline["pipeline"].(map[string]any)
	if !ok {
		return fmt.Errorf("pipeline JSON must contain a 'pipeline' object with nodes, values, and connection maps")
	}

	c := client.New(cfg.BaseURL, cfg.APIKey)

	if id != "" {
		fmt.Fprintf(stderr, "Updating pipeline... ")
		result, err := c.UpdatePipeline(id, inner, expectedVersion)
		if err != nil {
			fmt.Fprintln(stderr)
			return err
		}
		fmt.Fprintln(stderr, tui.SymbolCheck)

		if asJSON {
			return output.PrintJSON(result)
		}
		output.PrintUpdateResult(result)
		return nil
	}

	fmt.Fprintf(stderr, "Saving pipeline... ")
	result, err := c.SavePipeline(name, description, inner)
	if err != nil {
		fmt.Fprintln(stderr)
		return err
	}

	fmt.Fprintln(stderr, tui.SymbolCheck)

	if asJSON {
		return output.PrintJSON(result)
	}

	output.PrintSaveResult(result)
	return nil
}
