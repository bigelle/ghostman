/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package httpcmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/bigelle/ghostman/internal/httpcore"
	"github.com/bigelle/ghostman/internal/shared"
	"github.com/spf13/cobra"
)

type ctxKey string

const ctxKeyHttpReq ctxKey = "httpReq"

// HttpCmd represents the http command
var HttpCmd = &cobra.Command{
	Use:     "http",
	Short:   "deez nuts",
	Args:    cobra.ExactArgs(1),
	PreRunE: preHandleHttp,
	RunE:    handleHttp,
}

func init() {
	HttpCmd.PersistentFlags().Bool("dump-request", false, "dump the whole request")
	HttpCmd.PersistentFlags().Bool("dump-response", false, "dump the whole response")
	HttpCmd.PersistentFlags().Bool("send-request", true, "send request")
	HttpCmd.PersistentFlags().Bool("sanitize-cookies", true, "omits empty or malformed cookies")
	HttpCmd.PersistentFlags().Bool("sanitize-headers", true, "omits empty or malformed headers")
	HttpCmd.PersistentFlags().Bool("sanitize-query", true, "omits empty or malformed query parameters")

	HttpCmd.PersistentFlags().String(
		"data-json",
		"",
		"sets Content-Type header to 'application/json' and adds passed string as a body",
	)
	HttpCmd.PersistentFlags().String(
		"data-plain",
		"",
		"sets Content-Type header to 'text/plain' and adds passed string as a body",
	)
	HttpCmd.PersistentFlags().String(
		"data-html",
		"",
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)
	HttpCmd.PersistentFlags().StringArray(
		"data-form",
		[]string{},
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)
	HttpCmd.PersistentFlags().StringArray(
		"data-multipart",
		[]string{},
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)
}

func preHandleHttp(cmd *cobra.Command, args []string) error {
	path := args[0]
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	buf := shared.BytesBufPool.Get().(*bytes.Buffer)
	if _, err = buf.ReadFrom(file); err != nil {
		return err
	}
	req, err := httpcore.NewHttpRequestFromJSON(buf.Bytes())
	if err != nil {
		return err
	}
	buf.Reset()
	shared.BytesBufPool.Put(buf)

	applyRunTimeFlags(cmd, req)
	if isDataFlagUsed(cmd) {
		if err := applyBody(cmd, req); err != nil {
			return err
		}
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, ctxKeyHttpReq, *req)
	cmd.SetContext(withVal)
	return nil
}

func handleHttp(cmd *cobra.Command, args []string) error {
	builder := strings.Builder{}

	val := cmd.Context().Value(ctxKeyHttpReq)
	req, ok := val.(httpcore.HttpRequest)
	if !ok {
		return fmt.Errorf("can't read http request")
	}

	r, err := req.ToHTTP()
	if err != nil {
		return err
	}

	if req.ShouldDumpRequest {
		var d []byte
		d, err = dumpRequestSafely(r)
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
