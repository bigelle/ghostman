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

// HttpCmd represents the http command
var HttpCmd = &cobra.Command{
	Use:     "http",
	Short:   "deez nuts",
	Args:    cobra.ExactArgs(1),
	PreRunE: preHandleHttp, // TODO:
	RunE:    handleHttp,
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// httpCmd.PersistentFlags().String("foo", "", "A help for foo")
	HttpCmd.PersistentFlags().Bool("dump-request", false, "dump the whole request")
	HttpCmd.PersistentFlags().Bool("dump-response", false, "dump the whole response")
	HttpCmd.PersistentFlags().Bool("send-request", true, "send request")
	// TODO: add other flags for sanitizing empty cookies, headers, query

	// different body flags
	HttpCmd.PersistentFlags().String(
		"data-json",
		"",
		"sets Content-Type header to 'application/json' and adds passed string as a body",
	)
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// httpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func preHandleHttp(cmd *cobra.Command, args []string) error {
	path := args[0]
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer([]byte{})
	if _, err := buf.ReadFrom(file); err != nil {
		return err
	}
	req, err := httpcore.NewHttpRequestFromJSON(buf.Bytes())
	if err != nil {
		return err
	}
	applyRunTimeFlags(cmd, req)
	json, _ := cmd.Flags().GetString("data-json")
	if json != "" {
		if strings.HasPrefix(json, "@") {
			// treating like a file
			path := strings.TrimPrefix(json, "@")
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return fmt.Errorf("can't use a dir as a json")
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := req.SetBodyJSON(b); err != nil {
				return err
			}
		} else {
			// trying to treat it like a json
			b := []byte(json)
			if !httpcore.IsValidJSON(b) {
				return fmt.Errorf("not a valid json")
			}
			if err := req.SetBodyJSON(b); err != nil {
				return err
			}
		}
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, "httpReq", *req)
	cmd.SetContext(withVal)
	return nil
}

func handleHttp(cmd *cobra.Command, args []string) error {
	builder := strings.Builder{}

	val := cmd.Context().Value("httpReq")
	req, ok := val.(httpcore.HttpRequest)
	if !ok {
		return fmt.Errorf("can't read http request")
	}

	r, err := req.ToHTTP()
	if err != nil {
		return err
	}

	if req.ShouldDumpRequest {
		d, err := dumpRequestSafely(r)
		if err != nil {
			return err
		}
		builder.Write(d)
	}

	var resp *http.Response
	if req.ShouldSendRequest {
		client := shared.HttpClientPool.Get().(*http.Client)
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
		shared.HttpClientPool.Put(client)
	}

	fmt.Print(builder.String())
	return nil
}
