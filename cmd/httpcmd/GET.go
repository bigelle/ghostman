package httpcmd

import (
	"fmt"
	"io"
	"net/http/httputil"
	"strings"

	"github.com/spf13/cobra"
)

// is a child of http command
var getCmd = &cobra.Command{
	Use:     "GET",
	Short:   "send a GET request",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseCommand,
	RunE:    handleGET,
}

func init() {
	HttpCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// GETCmd.PersistentFlags().String("foo", "", "A help for foo")
	getCmd.Flags().StringArrayVarP(
		&headers,
		"header",
		"H",
		[]string{},
		"add a header to the request in format HeaderName:value.",
	)
	getCmd.Flags().StringArrayVarP(
		&query,
		"query",
		"Q",
		[]string{},
		"explicitly add a query parameter to the request URL in format QueryParam:value.",
	)
	getCmd.Flags().StringArrayVarP(
		&cookies,
		"cookie",
		"C",
		[]string{},
		"add a cookie to the request in format CookieName:value.",
		)

	// different body flags
	getCmd.Flags().StringVar(
		&jsonBody,
		"data-json",
		"",
		"sets Content-Type header to 'application/json' and adds passed string as a body",
	)

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// GETCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func handleGET(cmd *cobra.Command, args []string) error {
	val := cmd.Context().Value("httpReq")
	httpRequest, ok := val.(HttpRequest); if !ok {
		return fmt.Errorf("failed to get HTTP request from context")
	}

	builder := strings.Builder{}

	req, err := httpRequest.Request()
	if err != nil {
		return err
	}

	if httpRequest.ShouldDumpRequest {
		dump, err := dumpRequestSafely(req)
		if err != nil {
			return err
		}
		builder.Write(dump)
	}

	if httpRequest.ShouldSendRequest {
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if httpRequest.ShouldDumpResponse {
			dump, err := httputil.DumpResponse(resp, true)
			if err != nil {
				return err
			}
			builder.Write(dump)
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
