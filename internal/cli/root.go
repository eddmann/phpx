package cli

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "phpx",
	Short: "Run PHP scripts with inline dependencies",
	Long: `phpx runs PHP scripts with inline Composer dependencies and
executes Composer tools ephemerally.

Examples:
  phpx script.php              Run a PHP script
  phpx run script.php          Same as above
  phpx tool phpstan            Run PHPStan
  phpx tool phpstan@1.10.0     Run specific version`,
	SilenceUsage:  true,
	SilenceErrors: true,
	// Default: treat first arg as script if it's a .php file
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		// If first arg looks like a PHP file or is "-", run it
		if args[0] == "-" || isPhpFile(args[0]) {
			return runScript(cmd, args)
		}
		return cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show detailed output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress phpx output")
}

func Execute() error {
	return rootCmd.Execute()
}

func isPhpFile(path string) bool {
	return len(path) > 4 && path[len(path)-4:] == ".php"
}
