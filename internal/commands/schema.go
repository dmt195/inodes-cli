package commands

import (
	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Show available node types and their parameters",
		RunE:  runSchema,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runSchema(cmd *cobra.Command, args []string) error {
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

	c := client.New(cfg.BaseURL, cfg.APIKey)
	schema, err := c.GetSchemaNodes()
	if err != nil {
		return err
	}

	if asJSON {
		return output.PrintJSON(schema)
	}

	output.PrintNodeSchemas(schema)
	return nil
}
