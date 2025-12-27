package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newMCPCallCmd() *cobra.Command {
	var (
		argsJSON string
		argsFile string
	)

	cmd := &cobra.Command{
		Use:     "call <tool-name> [args-json]",
		Aliases: []string{"invoke"},
		Short:   "Call any MCP tool by name",
		Long: `Call any tool exposed by the Notion MCP server.

Use this for newly released MCP tools before dedicated CLI wrappers are added.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolName := strings.TrimSpace(args[0])
			if toolName == "" {
				return fmt.Errorf("tool name is required")
			}

			inlineArgs := argsJSON
			if len(args) == 2 {
				if strings.TrimSpace(argsJSON) != "" || strings.TrimSpace(argsFile) != "" {
					return fmt.Errorf("provide arguments via positional [args-json] OR --args/--args-file, not both")
				}
				inlineArgs = args[1]
			}

			toolArgs, err := parseMCPJSONObjectFromInlineOrFile(inlineArgs, argsFile, "args", "args-file")
			if err != nil {
				return err
			}
			if toolArgs == nil {
				toolArgs = map[string]interface{}{}
			}

			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.CallTool(ctx, toolName, toolArgs)
			if err != nil {
				return err
			}

			var parsed interface{}
			if err := json.Unmarshal([]byte(result), &parsed); err == nil {
				return printerForContext(ctx).Print(ctx, parsed)
			}
			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&argsJSON, "args", "", "Tool arguments as a JSON object")
	cmd.Flags().StringVar(&argsFile, "args-file", "", "Path to file containing tool arguments JSON")

	return cmd
}
