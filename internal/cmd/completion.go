package cmd

import "github.com/spf13/cobra"

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for ntn CLI.

To load completions:

Bash:
  $ source <(ntn completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ ntn completion bash > /etc/bash_completion.d/ntn
  # macOS:
  $ ntn completion bash > $(brew --prefix)/etc/bash_completion.d/ntn

Zsh:
  $ source <(ntn completion zsh)
  # To load completions for each session, execute once:
  $ ntn completion zsh > "${fpath[1]}/_ntn"

Fish:
  $ ntn completion fish | source
  # To load completions for each session, execute once:
  $ ntn completion fish > ~/.config/fish/completions/ntn.fish

PowerShell:
  PS> ntn completion powershell | Out-String | Invoke-Expression
  # To load completions for each session, execute once:
  PS> ntn completion powershell > ntn.ps1
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

	$ source <(ntn completion bash)

To load completions for every new session, execute once:

Linux:
	$ ntn completion bash > /etc/bash_completion.d/ntn

macOS:
	$ ntn completion bash > $(brew --prefix)/etc/bash_completion.d/ntn

You will need to start a new shell for this setup to take effect.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(stdoutFromContext(cmd.Context()))
		},
	}
}

func newCompletionZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate the autocompletion script for zsh.

To load completions in your current shell session:

	$ source <(ntn completion zsh)

To load completions for every new session, execute once:

	$ ntn completion zsh > "${fpath[1]}/_ntn"

You will need to start a new shell for this setup to take effect.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(stdoutFromContext(cmd.Context()))
		},
	}
}

func newCompletionFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: `Generate the autocompletion script for fish.

To load completions in your current shell session:

	$ ntn completion fish | source

To load completions for every new session, execute once:

	$ ntn completion fish > ~/.config/fish/completions/ntn.fish

You will need to start a new shell for this setup to take effect.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(stdoutFromContext(cmd.Context()), true)
		},
	}
}

func newCompletionPowershellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "powershell",
		Short: "Generate powershell completion script",
		Long: `Generate the autocompletion script for powershell.

To load completions in your current shell session:

	PS> ntn completion powershell | Out-String | Invoke-Expression

To load completions for every new session, execute once:

	PS> ntn completion powershell > ntn.ps1

and source this file from your PowerShell profile.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(stdoutFromContext(cmd.Context()))
		},
	}
}
