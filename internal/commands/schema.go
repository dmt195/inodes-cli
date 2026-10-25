package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema [node-type]",
		Short: "Show available node types and their parameters",
		Long: `Show available node types, or one node type in detail.

With a node type (for example "paragraph" or "NodeType.paragraph"), shows its
inputs, outputs and parameters, including defaults, ranges and allowed values.
--json prints the full schema, including allowed values ("enum").`,
		Args: cobra.MaximumNArgs(1),
		RunE: runSchema,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

// NewFontsCmd lists the fonts text and paragraph nodes accept.
func NewFontsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fonts",
		Short: "List the fonts available to text and paragraph nodes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			schema, err := fetchSchema(cmd)
			if err != nil {
				return err
			}
			node, err := findNodeSchema(schema.Nodes, "paragraph")
			if err != nil {
				return err
			}
			for _, p := range node.Parameters {
				if p.Name == "params_font" {
					if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
						return output.PrintJSON(p.Enum)
					}
					fmt.Println(strings.Join(p.Enum, "\n"))
					return nil
				}
			}
			return fmt.Errorf("the server's paragraph schema doesn't list fonts")
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func fetchSchema(cmd *cobra.Command) (*client.SchemaNodesResponse, error) {
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
	return client.New(cfg.BaseURL, cfg.APIKey).GetSchemaNodes()
}

func runSchema(cmd *cobra.Command, args []string) error {
	asJSON, _ := cmd.Flags().GetBool("json")

	schema, err := fetchSchema(cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		node, err := findNodeSchema(schema.Nodes, args[0])
		if err != nil {
			return err
		}
		if asJSON {
			return output.PrintJSON(node)
		}
		output.PrintNodeSchema(node)
		return nil
	}

	if asJSON {
		return output.PrintJSON(schema)
	}

	output.PrintNodeSchemas(schema)
	return nil
}

// findNodeSchema finds a node type by its full type ("NodeType.paragraph") or
// short name ("paragraph"), ignoring case.
func findNodeSchema(nodes []client.NodeSchema, name string) (*client.NodeSchema, error) {
	want := strings.ToLower(strings.TrimPrefix(strings.ToLower(name), "nodetype."))
	for i, n := range nodes {
		if strings.ToLower(strings.TrimPrefix(n.Type, "NodeType.")) == want {
			return &nodes[i], nil
		}
	}
	names := make([]string, 0, len(nodes))
	for _, n := range nodes {
		names = append(names, strings.TrimPrefix(n.Type, "NodeType."))
	}
	sort.Strings(names)
	return nil, fmt.Errorf("unknown node type %q; available: %s", name, strings.Join(names, ", "))
}
