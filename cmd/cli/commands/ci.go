package commands

import (
	"github.com/ChristofferNissen/helmper/internal"
	"github.com/spf13/cobra"
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Run an opinionated flow of Helmper with one configuration file as input",
	Run: func(cmd *cobra.Command, args []string) {
		internal.Program(args)
	},
}

func init() {
	rootCmd.AddCommand(ciCmd)
}
