/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cli

import (
	"fmt"
	"os"
	"slices"

	"github.com/bigelle/ghostman/cli/httpcmd"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "ghostman",
	Short: "deez nuts",
	Run: func(cmd *cobra.Command, args []string) { 
		if len(args) == 0 || slices.Contains(args, "--gui") {
			//TODO: probably should catch it in the main block
			fmt.Println("unimplemented")
		}
	},
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ghostman.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.AddCommand(httpcmd.HttpCmd)
}


