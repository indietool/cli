package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	RunE: func(cmd *cobra.Command, args []string) error {
		v := GetVersion()
		if jsonOutput {
			return printJSON(map[string]string{"version": v})
		}
		fmt.Println(v)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
