/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package httpcmd

import (
	"fmt"
	"io"
	"net/http/httputil"
	"strings"

	"github.com/bigelle/ghostman/internal/httpcore"
	"github.com/spf13/cobra"
)

// postCmd represents the POST command
var postCmd = &cobra.Command{
	Use:     "POST",
	Short:   "send a POST request",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseCommand,
	RunE:    handlePOST,
}

func init() {
	HttpCmd.AddCommand(postCmd)

	postCmd.Flags().StringArrayP(
		"header",
		"H",
		[]string{},
		"add a header to the request in format HeaderName:value.",
	)
	postCmd.Flags().StringArrayP(
		"query",
		"Q",
		[]string{},
		"explicitly add a query parameter to the request URL in format QueryParam:value.",
	)
	postCmd.Flags().StringArrayP(
		"cookie",
		"C",
		[]string{},
		"add a cookie to the request in format CookieName:value.",
	)
}

func handlePOST(cmd *cobra.Command, args []string) error {
	val := cmd.Context().Value(ctxKeyHttpReq)
	httpRequest, ok := val.(httpcore.HttpRequest)
	if !ok {
		return fmt.Errorf("failed to get HTTP request from context")
	}

	builder := strings.Builder{}

	req, err := httpRequest.ToHTTP()
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
