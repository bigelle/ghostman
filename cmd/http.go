package cmd

import (
	"context"
	"fmt"
	"io"
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
		return fmt.Errorf("error in pre-run task for http: %w", err)
	}

	ApplyRunTimeFlags(cmd, req)

	req, err = ApplyRequestFlags(cmd, *req)
	if err != nil {
		return fmt.Errorf("error in pre-run task for http: %w", err)
	}

	if HasAttachments(cmd) {
		if err := AttachBody(cmd, req); err != nil {
			return fmt.Errorf("error in pre-run task for http: %w", err)
		}
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, ctxKeyHttpReq, req)
	cmd.SetContext(withVal)

	return nil
}

func RunHttp(req *httpcore.Request) (err error) {
	sendReq := *req.Options.SendRequest

	str, err := req.ToString()
	if err != nil {
		return fmt.Errorf("error in run task for http: %w", err)
	}
	fmt.Println(str)

	var resp *httpcore.Response
	if sendReq {
		client := httpcore.NewClient()
		resp, err = client.Send(req)
		if err != nil {
			return fmt.Errorf("error in run task for http: %w", err)
		}
	}

	if !sendReq {
		return nil
	}

	fmt.Println(resp.String())

	if req.Options.Out != "" {
		if req.Options.Out == "stdout" {
			fmt.Println(resp.Body())
		}
	}
	return nil
}

func PreRunHttpFile(cmd *cobra.Command, args []string) error {
	buf := shared.BytesBuf()
	defer shared.PutBytesBuf(buf)

	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("can't read a request from a file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(buf, f)
	if err != nil {
		return err
	}

	req, err := httpcore.NewRequestFromJSON(buf.Bytes())
	if err != nil {
		return fmt.Errorf("malformed or invalid request file: %w", err)
	}

	ApplyRunTimeFlags(cmd, req)

	req, err = ApplyRequestFlags(cmd, *req)
	if err != nil {
		return fmt.Errorf("can't parse request flags: %w", err)

	}

	if HasAttachments(cmd) {
		if err := AttachBody(cmd, req); err != nil {
			return fmt.Errorf("can't attach body: %w", err)
		}
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, ctxKeyHttpReq, req)
	cmd.SetContext(withVal)

	return nil
}

func ApplyRunTimeFlags(cmd *cobra.Command, req *httpcore.Request) {
	if cmd.Flags().Changed("verbose") {
		f, _ := cmd.Flags().GetBool("verbose")
		req.Options.Verbose = &f
	}
	if cmd.Flags().Changed("out") {
		f, _ := cmd.Flags().GetString("out")
		req.Options.Out = f
	}
	if cmd.Flags().Changed("send-request") {
		f, _ := cmd.Flags().GetBool("send-request")
		req.Options.SendRequest = &f
	}
	if cmd.Flags().Changed("sanitize-cookies") {
		f, _ := cmd.Flags().GetBool("sanitize-cookies")
		req.Options.SanitizeCookies = &f
	}
	if cmd.Flags().Changed("sanitize-headers") {
		f, _ := cmd.Flags().GetBool("sanitize-headers")
		req.Options.SanitizeHeaders = &f
	}
	if cmd.Flags().Changed("sanitize-query") {
		f, _ := cmd.Flags().GetBool("sanitize-query")
		req.Options.SanitizeQuery = &f
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
			return nil, fmt.Errorf("invalid header: %w", err)
		}
		for k, v := range headers {
			req.AddHeader(k, v...)
		}
	}

	if cmd.Flags().Changed("query") {
		q, _ := cmd.Flags().GetStringArray("query")
		query, err := ParseKeyValues(q)
		if err != nil {
			return nil, fmt.Errorf("invalid query parameter: %w", err)
		}
		for k, v := range query {
			req.AddQueryParam(k, v...)
		}
	}

	if cmd.Flags().Changed("cookie") {
		c, _ := cmd.Flags().GetStringArray("cookie")
		cookies, err := ParseKeySingleValue(c)
		if err != nil {
			return nil, fmt.Errorf("invalid cookie: %w", err)
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
			return nil, fmt.Errorf("key-value pair must contain only one ':' separator: %s", raw)
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

	buf := shared.BytesBuf()
	defer shared.PutBytesBuf(buf)

	if strings.HasPrefix(arg, "@") {
		path := strings.TrimPrefix(arg, "@")

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}
		defer f.Close()

		_, err = io.Copy(buf, f)
		if err != nil {
			return fmt.Errorf("error reading content: %w", err)
		}

		ct := mimetype.Detect(buf.Bytes())

		body := httpcore.BodyGeneric{Ct: ct.String(), B: buf.Bytes()}

		req.SetBody(body)
	} else {
		buf.WriteString(arg)

		ct := mimetype.Detect(buf.Bytes())

		body := httpcore.BodyGeneric{Ct: ct.String(), B: buf.Bytes()}

		req.SetBody(body)
	}

	return nil
}

func AttachBodyForm(cmd *cobra.Command, req *httpcore.Request) error {
	args, _ := cmd.Flags().GetStringArray("form")
	body := httpcore.BodyForm{}

	for _, arg := range args {
		arg = strings.TrimSpace(arg)

		//FIXME: probably shouldn't use cut
		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			return fmt.Errorf("wrong form syntax: must be exactly one '=' separator")
		}

		if strings.HasPrefix(val, "@") {
			buf := shared.BytesBuf()
			defer shared.PutBytesBuf(buf)

			path := strings.TrimPrefix(val, "@")
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("error opening file: %w", err)
			}
			defer f.Close()

			_, err = io.Copy(buf, f)
			if err != nil {
				return fmt.Errorf("error reading content: %w", err)
			}

			val = buf.String()
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
			return fmt.Errorf("wrong part syntax: must be exactly one '=' separator")
		}

		if strings.HasPrefix(val, "@") {
			buf := shared.BytesBuf()
			defer shared.PutBytesBuf(buf)

			path := strings.TrimPrefix(val, "@")
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("error opening file: %w", err)
			}
			defer f.Close()

			_, err = io.Copy(buf, f)
			if err != nil {
				return fmt.Errorf("error reading content: %w", err)
			}

			if err := body.AddFile(key, val, buf.Bytes()); err != nil {
				return fmt.Errorf("error adding file part: %w", err)
			}
		} else if strings.HasPrefix(val, "<@") {
			buf := shared.BytesBuf()
			defer shared.PutBytesBuf(buf)

			path := strings.TrimPrefix(val, "<@")
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("error opening file: %w", err)
			}
			defer f.Close()

			_, err = io.Copy(buf, f)
			if err != nil {
				return fmt.Errorf("error reading content: %w", err)
			}

			if err := body.AddField(key, buf.String()); err != nil {
				return fmt.Errorf("error adding text part: %w", err)
			}
		} else {
			if err := body.AddField(key, val); err != nil {
				return fmt.Errorf("error adding text part: %w", err)
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
