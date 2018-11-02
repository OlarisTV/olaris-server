package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use: "olaris",
	Run: func(cmd *cobra.Command, args []string) {
		// Let root without arguments be an alias for serve:w
		serveCmd.Run(cmd, args)
	},
}

// Execute
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
