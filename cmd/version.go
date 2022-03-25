package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"simple-ec2/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of this binary",
	Long:  `Print the version number of this binary`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version %s", version.BuildInfo)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
