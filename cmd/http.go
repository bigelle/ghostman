package cmd

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
	"github.com/gabriel-vasile/mimetype"
	"github.com/spf13/cobra"
)

type ctxKey string

const ctxKeyHttpReq ctxKey = "httpReq"

func PreRunHttp(cmd *cobra.Command, args []string) error {
	req, err := httpcore.NewRequest(args[0])
	if err != nil {
		return err
	}
	ApplyRunTimeFlags(cmd, req)
	req, err = ApplyRequestFlags(cmd, *req)
	if err != nil {
		return err
	}
	if HasAttachments(cmd) {
		if err := AttachBody(cmd, req); err != nil {
			return err
		}
	}
	ctx := cmd.Context()
	withVal := context.WithValue(ctx, ctxKeyHttpReq, req)
	cmd.SetContext(withVal)
	return nil
}

func RunHttp(req *httpcore.Request) error {
	r, err := req.ToHTTP()
	if err != nil {
		return err
	}

	buf := shared.BytesBuf()
	defer shared.PutBytesBuf(buf)

	type httpRequest struct {
		resp *http.Response
		err  error
	}
	chResp := make(chan httpRequest, 1)

	if req.ShouldSendRequest {
		go func() {
			var resp *http.Response
			resp, err = httpcore.Client().Do(r)
			chResp <- httpRequest{resp: resp, err: err}
		}()
	}

	if req.ShouldDumpRequest {
		err = DumpRequestSafely(r, buf)
		if err != nil {
			return err
		}
	}

	if !req.ShouldSendRequest {
		// early exit
		fmt.Print(buf.String())
		return nil
	}

	result := <-chResp
	if result.err != nil {
		return err
	}

	if req.ShouldDumpResponse {
		err = DumpResponseSafely(result.resp, buf)
		if err != nil {
			return err
		}
	} else {
		b, err := io.ReadAll(result.resp.Body)
		if err != nil {
			return err
		}
		buf.Write(b)
		if !bytes.HasSuffix(b, []byte("\n")) {
			buf.WriteString("\n")
		}
	}
	fmt.Print(buf.String())
	return nil
}

func PreRunHttpFile(cmd *cobra.Command, args []string) error {
	b, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	req, err := httpcore.NewRequestFromJSON(b)
	if err != nil {
		return err
	}
	ApplyRunTimeFlags(cmd, req)
	req, err = ApplyRequestFlags(cmd, *req)
	if err != nil {
		return err
	}
	if HasAttachments(cmd) {
		if err := AttachBody(cmd, req); err != nil {
			return err
		}
	}
	ctx := cmd.Context()
	withVal := context.WithValue(ctx, ctxKeyHttpReq, req)
	cmd.SetContext(withVal)
	return nil
}

func DumpRequestSafely(req *http.Request, w *bytes.Buffer) error {
	clone, err := CloneRequest(req)
	if err != nil {
		return err
	}
	b, err := httputil.DumpRequestOut(clone, true)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	if !bytes.HasSuffix(b, []byte("\n")) {
		w.WriteString("\n")
	}
	return err
}

func CloneRequest(req *http.Request) (*http.Request, error) {
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

func DumpResponseSafely(resp *http.Response, w *bytes.Buffer) error {
	var bodyBytes []byte
	if resp.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	clone := *resp
	if bodyBytes != nil {
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	b, err := httputil.DumpResponse(&clone, true)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	if !bytes.HasSuffix(b, []byte("\n")) {
		w.WriteString("\n")
	}
	return err
}

func ApplyRunTimeFlags(cmd *cobra.Command, req *httpcore.Request) {
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

func ApplyRequestFlags(cmd *cobra.Command, req httpcore.Request) (*httpcore.Request, error) {
	if cmd.Flags().Changed("method") {
		m, _ := cmd.Flags().GetString("method")
		req.Method = m
	}

	if cmd.Flags().Changed("header") {
		h, _ := cmd.Flags().GetStringArray("header")
		headers, err := ParseKeyValues(h)
		if err != nil {
			return nil, err
		}
		for k, v := range headers {
			req.AddHeader(k, v...)
		}
	}

	if cmd.Flags().Changed("query") {
		q, _ := cmd.Flags().GetStringArray("query")
		query, err := ParseKeyValues(q)
		if err != nil {
			return nil, err
		}
		for k, v := range query {
			req.AddQueryParam(k, v...)
		}
	}

	if cmd.Flags().Changed("cookie") {
		c, _ := cmd.Flags().GetStringArray("cookie")
		cookies, err := ParseKeySingleValue(c)
		if err != nil {
			return nil, err
		}
		for k, v := range cookies {
			req.AddCookie(k, v)
		}
	}

	return &req, nil
}

// works with both headers and query parameters
func ParseKeyValues(h []string) (map[string][]string, error) {
	// example: -H "Accept:application/json,text/plain"
	// should return: "Accept": {"application/json", "text/plain"}
	// same with query params
	result := make(map[string][]string)
	for _, raw := range h {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("wrong key:value pair format: %s", raw)
		}
		key := strings.TrimSpace(parts[0])
		values := strings.Split(parts[1], ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		result[key] = append(result[key], values...)
	}
	return result, nil
}

func ParseKeySingleValue(h []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, raw := range h {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("wrong key:value pair format: %s", raw)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = value
	}
	return result, nil
}

func AttachBody(cmd *cobra.Command, req *httpcore.Request) error {
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

func AttachBodyData(cmd *cobra.Command, req *httpcore.Request) error {
	arg, _ := cmd.Flags().GetString("data")
	arg = strings.TrimSpace(arg)

	if strings.HasPrefix(arg, "@") {
		path := strings.TrimPrefix(arg, "@")
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ct := mimetype.Detect(b)
		body := httpcore.BodyGeneric{Ct: ct.String(), B: b}
		req.SetBody(body)
	} else {
		b := []byte(arg)
		ct := mimetype.Detect(b)
		body := httpcore.BodyGeneric{Ct: ct.String(), B: b}
		req.SetBody(body)
	}
	return nil
}

func AttachBodyForm(cmd *cobra.Command, req *httpcore.Request) error {
	args, _ := cmd.Flags().GetStringArray("form")
	body := httpcore.BodyForm{}

	for _, arg := range args {
		arg = strings.TrimSpace(arg)

		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			return fmt.Errorf("wrong form syntax")
		}
		if strings.HasPrefix(val, "@") {
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

func AttachBodyMultipart(cmd *cobra.Command, req *httpcore.Request) error {
	args, _ := cmd.Flags().GetStringArray("part")
	body := httpcore.NewBodyMultipart()

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
		} else if strings.HasPrefix(val, "<@") {
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
