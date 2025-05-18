/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package httpcmd

import (
	"github.com/spf13/cobra"
)

// HttpCmd represents the http command
var HttpCmd = &cobra.Command{
	Use:   "http",
	Short: "deez nuts",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// httpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// httpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
