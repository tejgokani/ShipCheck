package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print shipcheck version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("shipcheck", cmd.Root().Version)
	},
}
