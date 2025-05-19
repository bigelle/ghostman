package httpcmd

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/spf13/cobra"
)

// is a child of http command
var getCmd = &cobra.Command{
	Use:   "GET",
	Short: "send a GET request",
	Args:  cobra.ExactArgs(1),
	RunE:  handleGET,
}

func init() {
	HttpCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// GETCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// GETCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func handleGET(cmd *cobra.Command, args []string) error {
	url := args[0]
	client := http.DefaultClient

	resp, err := client.Get(url)
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
