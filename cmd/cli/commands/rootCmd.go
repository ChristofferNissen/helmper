package commands

import (
	"fmt"
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/spf13/cobra"
)

var convenienceGroup = &cobra.Group{
	ID:    "convenience",
	Title: "Convenience commands",
}

var remoteGroup = &cobra.Group{
	ID:    "remote",
	Title: "Commands for working with remote registry",
}

var localGroup = &cobra.Group{
	ID:    "local",
	Title: "Commands for working with local files",
}

func init() {
	rootCmd.AddGroup(convenienceGroup)
	rootCmd.AddGroup(remoteGroup)
	rootCmd.AddGroup(localGroup)
}

var rootCmd = &cobra.Command{
	Use:   "helmper-cli",
	Short: "Helmper-cli is a very fast static site generator",
	Long:  figure.NewFigure("helmper", "rectangles", true).String(),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
