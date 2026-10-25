package commands

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "versions <pipeline-id>",
		Short:             "List a pipeline's saved versions",
		Long:              "List a pipeline's saved versions, newest first. Restore one with 'inodes restore'.",
		Args:              cobra.ExactArgs(1),
		RunE:              runVersions,
		ValidArgsFunction: completePipelineIDs,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func NewRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <pipeline-id> <version>",
		Short: "Restore a pipeline to an earlier version",
		Long: `Make an earlier version of a pipeline the current one. The restore is saved
as a new version, so it can be undone the same way.`,
		Args: cobra.ExactArgs(2),
		RunE: runRestore,
	}
}

func newClientFromFlags(cmd *cobra.Command) (*client.Client, error) {
	cfg, err := config.Load(
		cmd.Root().PersistentFlags().Lookup("api-key").Value.String(),
		cmd.Root().PersistentFlags().Lookup("base-url").Value.String(),
	)
	if err != nil {
		return nil, err
	}
	if err := cfg.RequireAPIKey(); err != nil {
		return nil, err
	}
	return client.New(cfg.BaseURL, cfg.APIKey), nil
}

func runVersions(cmd *cobra.Command, args []string) error {
	c, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}
	versions, err := c.ListVersions(args[0])
	if err != nil {
		return err
	}
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON || !output.IsInteractive() {
		return output.PrintJSON(versions)
	}
	if len(versions) == 0 {
		fmt.Println(tui.Muted.Render("No saved versions."))
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tui.Bold.Render("VERSION")+"\t"+tui.Bold.Render("SAVED")+"\t"+tui.Bold.Render("BY"))
	for _, v := range versions {
		fmt.Fprintf(w, "%d\t%s\t%s\n", v.Version, v.CreatedAt, v.SavedByEmail)
	}
	return w.Flush()
}

func runRestore(cmd *cobra.Command, args []string) error {
	version, err := strconv.Atoi(args[1])
	if err != nil || version < 1 {
		return fmt.Errorf("version must be a positive number, got %q", args[1])
	}
	c, err := newClientFromFlags(cmd)
	if err != nil {
		return err
	}
	newVersion, err := c.RestoreVersion(args[0], version)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s Restored version %d as version %d\n", tui.SymbolCheck, version, newVersion)
	return nil
}
