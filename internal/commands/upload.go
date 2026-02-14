package commands

import (
	"fmt"
	"os"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewUploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload an ephemeral image (expires in 24h)",
		Args:  cobra.ExactArgs(1),
		RunE:  runUpload,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runUpload(cmd *cobra.Command, args []string) error {
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

	filePath := args[0]
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}

	asJSON, _ := cmd.Flags().GetBool("json")

	c := client.New(cfg.BaseURL, cfg.APIKey)

	if !asJSON {
		fmt.Fprintf(os.Stderr, "Uploading %s... ", filePath)
	}

	result, err := c.UploadEphemeral(filePath)
	if err != nil {
		if !asJSON {
			fmt.Fprintln(os.Stderr, tui.SymbolCross)
		}
		return err
	}

	if asJSON {
		return output.PrintJSON(result)
	}

	fmt.Fprintln(os.Stderr, tui.SymbolCheck)
	fmt.Println(result.ID)
	fmt.Fprintf(os.Stderr, "%s Expires: %s\n", tui.Muted.Render("  "), result.ExpiresAt)
	return nil
}
