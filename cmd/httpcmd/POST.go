/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package httpcmd

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/spf13/cobra"
)

// postCmd represents the POST command
var postCmd = &cobra.Command{
	Use:   "POST",
	Short: "send a POST request",
	Args: cobra.ExactArgs(2),
	RunE: handlePOST,
}

var contentType = "text/plain"

func init() {
	HttpCmd.AddCommand(postCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// POSTCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// POSTCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	postCmd.Flags().StringVar(&contentType, "content-type", "text/plain", "set a content type for this request")
}

func handlePOST(cmd *cobra.Command, args []string) error {
	url := args[0]
	client := http.DefaultClient
	r, err := os.Open(args[1])
	if err != nil {
		return err
	}

	resp, err := client.Post(url, contentType, r)
	if err != nil {
		return err
	}

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}
	fmt.Println(string(dump))

	return nil
}
