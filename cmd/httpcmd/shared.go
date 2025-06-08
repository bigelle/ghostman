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
	"github.com/gabriel-vasile/mimetype"
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
	if HasAttachments(cmd) {
		if err := AttachBody(cmd, req); err != nil {
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

func AttachBody(cmd *cobra.Command, req *httpcore.HttpRequest) error {
	switch {
	case cmd.Flags().Changed("data"):
		return AttachBodyData(cmd, req)
	case cmd.Flags().Changed("form"):
		return AttachBodyForm(cmd, req)
	case cmd.Flags().Changed("part"):
	return AttachBodyMultipart(cmd, req)
	default:
		return fmt.Errorf("you messed up flags")
	}
}

func AttachBodyData(cmd *cobra.Command, req *httpcore.HttpRequest) error {
	arg, _ := cmd.Flags().GetString("data")
	arg = strings.TrimSpace(arg)

	if strings.HasPrefix(arg, "@") {
		path := strings.TrimPrefix(arg, "@")
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ct := mimetype.Detect(b)
		body := httpcore.NewHttpBodyGeneric(ct.String(), b)
		req.SetBody(body)
	} else {
		b := []byte(arg)
		ct := mimetype.Detect(b)
		body := httpcore.NewHttpBodyGeneric(ct.String(), b)
		req.SetBody(body)
	}
	return nil
}

func AttachBodyForm(cmd *cobra.Command, req *httpcore.HttpRequest) error {
	args, _ := cmd.Flags().GetStringArray("form")
	body := httpcore.HttpBodyForm{}
	
	for _, arg := range args {
		arg = strings.TrimSpace(arg)

		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			return fmt.Errorf("wrong form syntax")
		}
		if strings.HasPrefix(val,"@") {
			path := strings.TrimPrefix(val, "@")
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			val = string(b)
		}
		body.Add(key, val)
	}
	req.SetBody(body)
	return nil
}

func AttachBodyMultipart(cmd *cobra.Command, req *httpcore.HttpRequest) error {
	args, _ := cmd.Flags().GetStringArray("part")
	body := httpcore.NewHttpBodyMultipart()

	for _, arg := range args {
		arg = strings.TrimSpace(arg)

		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			return fmt.Errorf("wrong part syntax")
		}
		if strings.HasPrefix(val, "@") {
			path := strings.TrimPrefix(val, "@")
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := body.AddFile(key, val, b); err != nil {
				return err
			}
		} else  if strings.HasPrefix(val, "<@") {
			path := strings.TrimPrefix(val, "<@")
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := body.AddField(key, string(b)); err != nil {
				return err
			}
		} else {
			if err := body.AddField(key, val); err != nil {
				return err
			}
		}
	}
	req.SetBody(body)
	return nil
}

func HasAttachments(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("data") ||
		cmd.Flags().Changed("form") ||
		cmd.Flags().Changed("part")
}
