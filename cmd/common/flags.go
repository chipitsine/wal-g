package common

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagsCmd represents the flags command
var FlagsCmd = &cobra.Command{
	Use:                   "flags",
	Short:                 "Display the list of available global flags for all wal-g commands",
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Usage()
	},
}

func init() {
	FlagsCmd.SetUsageTemplate(flagsUsageTemplate)
	FlagsCmd.SetHelpTemplate(flagsHelpTemplate)

	// fix to disable the required settings check for the help subcommand
	FlagsCmd.PersistentPreRun = func(*cobra.Command, []string) {}

	defaultUsage := FlagsCmd.UsageFunc()
	FlagsCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		unhideGlobalFlags(cmd)
		return defaultUsage(cmd)
	})
}

func unhideGlobalFlags(cmd *cobra.Command) {
	if root := cmd.Root(); root != nil {
		root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			f.Hidden = false
		})
	}
}

const flagsHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}{{end}}

Usage:
{{.UseLine}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

{{.Usage}}`
const flagsUsageTemplate = `Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
`
