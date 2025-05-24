/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package httpcmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/spf13/cobra"
)

// HttpCmd represents the http command
var HttpCmd = &cobra.Command{
	Use:   "http",
	Short: "deez nuts",
	Args:  cobra.ExactArgs(1),
	PreRunE: readHttpFile,
	RunE:  handleHttp,
}

// flag values
var (
	shouldDumpRequest  bool
	shouldDumpResponse bool
	shouldSendRequest  bool
	headers            []string
	query              []string
	jsonBody           string
)

var client = http.DefaultClient

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// httpCmd.PersistentFlags().String("foo", "", "A help for foo")
	HttpCmd.PersistentFlags().BoolVar(&shouldDumpRequest, "dump-request", false, "dump the whole request")
	HttpCmd.PersistentFlags().BoolVar(&shouldDumpResponse, "dump-response", false, "dump the whole response")
	HttpCmd.PersistentFlags().BoolVar(&shouldSendRequest, "send-request", true, "send request")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// httpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func handleHttp(cmd *cobra.Command, args []string) error {
	builder := strings.Builder{}

	val := cmd.Context().Value("httpReq")
	req, ok := val.(HttpRequest); if !ok {
		return fmt.Errorf("can't read http request")
	}

	r, err := req.Request()
	if err != nil {
		return err
	}

	if req.ShouldDumpRequest {
		d, err := httputil.DumpRequest(r, true)
		if err != nil {
			return err
		}
		builder.Write(d)
	}

	var resp *http.Response
	if req.ShouldSendRequest {
		resp, err = client.Do(r)
		if err != nil {
			return err
		}
		if req.ShouldDumpResponse {
			d, err := httputil.DumpResponse(resp, true)
			if err != nil {
				return err
			}
			builder.Write(d)
		} else {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			builder.Write(b)
		}
	}

	fmt.Print(builder.String())
	return nil
}
