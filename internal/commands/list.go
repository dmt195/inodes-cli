package commands

import (
	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your pipelines",
		RunE:  runList,
	}
	cmd.Flags().Int("offset", 0, "Pagination offset")
	cmd.Flags().Int("page-size", 20, "Number of pipelines per page")
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
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

	offset, _ := cmd.Flags().GetInt("offset")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	asJSON, _ := cmd.Flags().GetBool("json")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	result, err := c.ListPipelines(offset, pageSize)
	if err != nil {
		return err
	}

	if asJSON {
		return output.PrintJSON(result)
	}

	output.PrintPipelineList(result)
	return nil
}
