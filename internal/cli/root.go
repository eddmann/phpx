package cli

import (
	"github.com/spf13/cobra"
)

const logo = `
 ██████╗ ██╗  ██╗██████╗ ██╗  ██╗
 ██╔══██╗██║  ██║██╔══██╗╚██╗██╔╝
 ██████╔╝███████║██████╔╝ ╚███╔╝
 ██╔═══╝ ██╔══██║██╔═══╝  ██╔██╗
 ██║     ██║  ██║██║     ██╔╝ ██╗
 ╚═╝     ╚═╝  ╚═╝╚═╝     ╚═╝  ╚═╝
`

var (
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:     "phpx",
	Short:   "Run PHP scripts with inline dependencies",
	Version: Version,
	Long: `phpx runs PHP scripts with inline Composer dependencies and
executes Composer tools ephemerally.

Examples:
  phpx script.php              Run a PHP script
  phpx run script.php          Same as above
  phpx tool phpstan            Run PHPStan
  phpx tool phpstan@1.10.0     Run specific version`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return runScript(cmd, args)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show detailed output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress phpx output")

	rootCmd.SetHelpTemplate(logo + `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`)
}

func Execute() error {
	return rootCmd.Execute()
}
