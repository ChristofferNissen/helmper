package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Helmper",
	Long:  `All software has versions. This is Helmper's`,
	Run: func(cmd *cobra.Command, args []string) {
		s := fmt.Sprintf("version %s (commit %s, built at %s)", version, commit, date)
		fmt.Println(s)
	},
}
