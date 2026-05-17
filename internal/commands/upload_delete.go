package commands

import (
	"fmt"
	"os"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewUploadDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload-delete <ephemeral-asset-id>",
		Short: "Delete an ephemeral asset before its 24h TTL expires",
		Long: `Delete an ephemeral asset uploaded via 'inodes upload'.

Ephemeral assets normally expire after 24 hours. Use this to free a slot
against the per-user limit before that — useful in CI runs that produce
more than the configured EPHEMERAL_ASSET_LIMIT in quick succession.

The ephemeral-asset-id is the 26-character ULID returned by 'inodes upload'.

Examples:
  inodes upload-delete 01KM2XGX2RPYRQ9F7V2ZP3F5TQ
  inodes upload-delete 01KM2XGX2RPYRQ9F7V2ZP3F5TQ --json`,
		Args: cobra.ExactArgs(1),
		RunE: runUploadDelete,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runUploadDelete(cmd *cobra.Command, args []string) error {
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
	assetID := args[0]

	if !asJSON {
		fmt.Fprintf(os.Stderr, "Deleting ephemeral asset %s... ", assetID)
	}

	c := client.New(cfg.BaseURL, cfg.APIKey)
	if err := c.DeleteEphemeral(assetID); err != nil {
		if !asJSON {
			fmt.Fprintln(os.Stderr, tui.SymbolCross)
		}
		return err
	}

	if asJSON {
		fmt.Printf(`{"deleted": "%s"}%s`, assetID, "\n")
		return nil
	}

	fmt.Fprintln(os.Stderr, tui.SymbolCheck)
	return nil
}
