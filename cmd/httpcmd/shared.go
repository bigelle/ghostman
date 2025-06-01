package httpcmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/bigelle/ghostman/internal/httpcore"
	"github.com/spf13/cobra"
)

var client = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		MaxConnsPerHost:     10,
		MaxIdleConnsPerHost: 10,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		Proxy:               http.ProxyFromEnvironment,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

func parseCommand(cmd *cobra.Command, args []string) error {
	req, err := httpcore.NewHttpRequest(args[0], cmd.Name())
	if err != nil {
		return err
	}
	applyRunTimeFlags(cmd, req)
	req, err = applyRequestFlags(cmd, *req)
	if err != nil {
		return err
	}
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

func dumpRequestSafely(req *http.Request) ([]byte, error) {
	clone, err := cloneRequest(req)
	if err != nil {
		return nil, err
	}
	return httputil.DumpRequestOut(clone, true)
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	clone := req.Clone(req.Context())
	if bodyBytes != nil {
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	return clone, nil
}

func applyRunTimeFlags(cmd *cobra.Command, req *httpcore.HttpRequest) {
	if cmd.Flags().Changed("dump-request") {
		f, _ := cmd.Flags().GetBool("dump-request")
		req.ShouldDumpRequest = f
	}
	if cmd.Flags().Changed("dump-response") {
		f, _ := cmd.Flags().GetBool("dump-response")
		req.ShouldDumpResponse = f
	}
	if cmd.Flags().Changed("send-request") {
		f, _ := cmd.Flags().GetBool("send-request")
		req.ShouldSendRequest = f
	}
	if cmd.Flags().Changed("sanitize-cookies") {
		f, _ := cmd.Flags().GetBool("sanitize-cookies")
		req.ShouldSanitizeCookies = f
	}
	if cmd.Flags().Changed("sanitize-headers") {
		f, _ := cmd.Flags().GetBool("sanitize-headers")
		req.ShouldSanitizeHeaders = f
	}
	if cmd.Flags().Changed("sanitize-query") {
		f, _ := cmd.Flags().GetBool("sanitize-query")
		req.ShouldDumpResponse = f
	}
}

func applyRequestFlags(cmd *cobra.Command, req httpcore.HttpRequest) (*httpcore.HttpRequest, error) {
	h, _ := cmd.Flags().GetStringArray("header")
	headers, err := httpcore.ParseKeyValues(h)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.AddHeader(k, v...)
	}

	q, _ := cmd.Flags().GetStringArray("query")
	query, err := httpcore.ParseKeyValues(q)
	if err != nil {
		return nil, err
	}
	for k, v := range query {
		req.AddQueryParam(k, v...)
	}

	c, _ := cmd.Flags().GetStringArray("cookie")
	cookies, err := httpcore.ParseKeySingleValue(c)
	if err != nil {
		return nil, err
	}
	for k, v := range cookies {
		req.AddCookie(k, v)
	}

	return &req, nil
}

func isDataFlagUsed(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("data-json") ||
		cmd.Flags().Changed("data-plain") ||
		cmd.Flags().Changed("data-html")
}

func applyBody(cmd *cobra.Command, req *httpcore.HttpRequest) error {
	switch {
	case cmd.Flags().Changed("data-json"):
		return applyBodyJSON(cmd, req)
	case cmd.Flags().Changed("data-plain"):
		return applyBodyPlainText(cmd, req)
	case cmd.Flags().Changed("data-html"):
		return applyBodyHTML(cmd, req)
	default:
		return nil
	}
}

func applyBodyJSON(cmd *cobra.Command, req *httpcore.HttpRequest) error {
	json, _ := cmd.Flags().GetString("data-json")
	json = strings.TrimSpace(json)
	if isFile(json) {
		// treating like a file
		path := strings.TrimPrefix(json, "@")
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
	return nil
}

func applyBodyPlainText(cmd *cobra.Command, req *httpcore.HttpRequest) error {
	arg, _ := cmd.Flags().GetString("data-plain")
	arg = strings.TrimSpace(arg)
	txt := arg
	enc := "utf-8"

	if i := strings.Index(txt, ":"); i != -1 {
		pref := strings.ToLower(arg[:i])
		switch pref {
		//TODO: other encodings
		case "utf-8", "utf-16":
			enc = pref
			txt = arg[i+1:]
		default:
			return fmt.Errorf("not a valid encoding")
		}
	}

	if isFile(txt) {
		file := strings.TrimPrefix(txt, "@")
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := req.SetBodyPlainText(b, enc); err != nil {
			return err
		}
	} else {
		b := []byte(txt)
		if err := req.SetBodyPlainText(b, enc); err != nil {
			return err
		}
	}
	return nil
}

func applyBodyHTML(cmd *cobra.Command, req *httpcore.HttpRequest) error {
	arg, _ := cmd.Flags().GetString("data-html")
	arg = strings.TrimSpace(arg)

	if isFile(arg) {
		file := strings.TrimPrefix(arg, "@")
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := req.SetBodyHTML(b); err != nil {
			return err
		}
	} else {
		b := []byte(arg)
		if err := req.SetBodyHTML(b); err != nil {
			return err
		}
	}
	return nil
}

func isFile(str string) bool {
	return strings.HasPrefix(str, "@")
}
