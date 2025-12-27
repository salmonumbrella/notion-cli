package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for notion CLI.

To load completions:

Bash:
  $ source <(notion completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ notion completion bash > /etc/bash_completion.d/notion
  # macOS:
  $ notion completion bash > $(brew --prefix)/etc/bash_completion.d/notion

Zsh:
  $ source <(notion completion zsh)
  # To load completions for each session, execute once:
  $ notion completion zsh > "${fpath[1]}/_notion"

Fish:
  $ notion completion fish | source
  # To load completions for each session, execute once:
  $ notion completion fish > ~/.config/fish/completions/notion.fish

PowerShell:
  PS> notion completion powershell | Out-String | Invoke-Expression
  # To load completions for each session, execute once:
  PS> notion completion powershell > notion.ps1
  # and source this file from your PowerShell profile.
`,
	}

	cmd.AddCommand(newCompletionBashCmd())
	cmd.AddCommand(newCompletionZshCmd())
	cmd.AddCommand(newCompletionFishCmd())
	cmd.AddCommand(newCompletionPowershellCmd())

	return cmd
}

func newCompletionBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long: `Generate the autocompletion script for bash.

To load completions in your current shell session:

	$ source <(notion completion bash)

To load completions for every new session, execute once:

Linux:
	$ notion completion bash > /etc/bash_completion.d/notion

macOS:
	$ notion completion bash > $(brew --prefix)/etc/bash_completion.d/notion

You will need to start a new shell for this setup to take effect.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	}
}

func newCompletionZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate the autocompletion script for zsh.

To load completions in your current shell session:

	$ source <(notion completion zsh)

To load completions for every new session, execute once:

	$ notion completion zsh > "${fpath[1]}/_notion"

You will need to start a new shell for this setup to take effect.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	}
}

func newCompletionFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: `Generate the autocompletion script for fish.

To load completions in your current shell session:

	$ notion completion fish | source

To load completions for every new session, execute once:

	$ notion completion fish > ~/.config/fish/completions/notion.fish

You will need to start a new shell for this setup to take effect.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	}
}

func newCompletionPowershellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "powershell",
		Short: "Generate powershell completion script",
		Long: `Generate the autocompletion script for powershell.

To load completions in your current shell session:

	PS> notion completion powershell | Out-String | Invoke-Expression

To load completions for every new session, execute once:

	PS> notion completion powershell > notion.ps1

and source this file from your PowerShell profile.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		},
	}
}
