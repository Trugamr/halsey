package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "halsey",
	Short: "Halsey is a tool for downloading HLS streams",
	Long:  "Halsey is a command line tool for downloading HLS streams from a given URL",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
