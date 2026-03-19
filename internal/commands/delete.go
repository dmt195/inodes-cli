package commands

import (
	"fmt"
	"os"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <pipeline-id>",
		Short: "Delete a pipeline",
		Long: `Delete a pipeline from your account. This action cannot be undone.

The pipeline-id is a 26-character ULID (e.g., 01KM2XGX2RPYRQ9F7V2ZP3F5TQ).

Examples:
  inodes delete 01KM2XGX2RPYRQ9F7V2ZP3F5TQ
  inodes delete 01KM2XGX2RPYRQ9F7V2ZP3F5TQ --force`,
		Args:              cobra.ExactArgs(1),
		RunE:              runDelete,
		ValidArgsFunction: completePipelineIDs,
	}
	cmd.Flags().Bool("force", false, "Skip confirmation prompt")
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	force, _ := cmd.Flags().GetBool("force")
	asJSON, _ := cmd.Flags().GetBool("json")
	pipelineID := args[0]

	if !force {
		fmt.Fprintf(os.Stderr, "Delete pipeline %s? This cannot be undone. [y/N] ", pipelineID)
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	fmt.Fprintf(os.Stderr, "Deleting pipeline... ")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	if err := c.DeletePipeline(pipelineID); err != nil {
		fmt.Fprintln(os.Stderr)
		return err
	}

	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if asJSON {
		fmt.Printf(`{"deleted": "%s"}%s`, pipelineID, "\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "%s Pipeline %s deleted\n", tui.SymbolCheck, pipelineID)
	return nil
}
