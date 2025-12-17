package cmd

import (
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	outputFormat output.Format

	// Version information
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "notion",
	Short: "CLI for Notion API",
	Long:  `A command-line interface for interacting with the Notion API`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Parse and validate output format from flag
		formatStr, _ := cmd.Flags().GetString("output")
		format, err := output.ParseFormat(formatStr)
		if err != nil {
			return err
		}
		outputFormat = format
		return nil
	},
}

func init() {
	// Set version info
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("notion-cli %s (commit: %s, built: %s)\n", version, commit, buildTime))

	// Global flags
	rootCmd.PersistentFlags().String("output", "text", "Output format (text|json|table)")

	// Register subcommands
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newUserCmd())
	rootCmd.AddCommand(newPageCmd())
	rootCmd.AddCommand(newBlockCmd())
	rootCmd.AddCommand(newDBCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newCommentCmd())
	rootCmd.AddCommand(newFileCmd())
	rootCmd.AddCommand(newDataSourceCmd())
}

// Execute runs the root command
func Execute(args []string) error {
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// GetOutputFormat returns the current output format for use by subcommands
func GetOutputFormat() output.Format {
	return outputFormat
}

// SetVersionInfo sets the version information for the CLI
func SetVersionInfo(v, c, b string) {
	version = v
	commit = c
	buildTime = b
	rootCmd.Version = v
	rootCmd.SetVersionTemplate(fmt.Sprintf("notion-cli %s (commit: %s, built: %s)\n", v, c, b))
}
