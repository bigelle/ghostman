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

const (
	ctxKeyHttpReq  ctxKey = "httpReq"
	ctxKeyHttpOpts ctxKey = "httpOpts"
)

type Options struct {
	SendRequest bool
	Verbose     bool
	Out         string
}

func PreRunHttp(cmd *cobra.Command, args []string) (err error) {
	reqURL, err := httpcore.DetectSchema(args[0])
	if err != nil {
		return fmt.Errorf("detecting URL schema: %w", err)
	}

	req, err := httpcore.NewRequest(reqURL)
	if err != nil {
		return fmt.Errorf("error in pre-run task for http: %w", err)
	}

	buf, ct, err := ParseAttachments(cmd)
	if err != nil {
		return fmt.Errorf("parsing attachments: %w", err)
	}

	if buf != nil && ct != "" {
		req.SetBody(buf, ct)
	}

	err = ApplyRequestFlags(cmd, req)
	if err != nil {
		return fmt.Errorf("error in pre-run task for http: %w", err)
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, ctxKeyHttpReq, req)

	opts := GetOptions(cmd)
	withVal = context.WithValue(withVal, ctxKeyHttpOpts, opts)

	cmd.SetContext(withVal)

	return nil
}

func RunHttp(cmd *cobra.Command, args []string) (err error) {
	opts := cmd.Context().Value(ctxKeyHttpOpts).(Options)
	req := cmd.Context().Value(ctxKeyHttpReq).(*httpcore.RequestConf)

	str, err := req.ToString()
	if err != nil {
		return fmt.Errorf("formatting request: %w", err)
	}
	fmt.Println(str)

	if !opts.SendRequest {
		return nil
	}

	client := httpcore.NewClient()
	resp, err := client.Send(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	str, err = resp.ToString()
	if err != nil {
		return fmt.Errorf("formatting response: %w", err)
	}
	fmt.Printf("\n%s\n", str)

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

	b, ct, err := ParseAttachments(cmd)
	if err != nil {
		return fmt.Errorf("parsing attachments: %w", err)
	}

	if b != nil && ct != "" {
		req.SetBody(b, ct)
	}

	err = ApplyRequestFlags(cmd, req)
	if err != nil {
		return fmt.Errorf("can't parse request flags: %w", err)
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, ctxKeyHttpReq, req)

	opts := GetOptions(cmd)
	withVal = context.WithValue(withVal, ctxKeyHttpOpts, opts)

	cmd.SetContext(withVal)

	return nil
}

func GetOptions(cmd *cobra.Command) Options {
	opts := Options{
		Verbose:     false,
		SendRequest: true,
		Out:         "stdout",
	}

	if cmd.Flags().Changed("verbose") {
		f, _ := cmd.Flags().GetBool("verbose")
		opts.Verbose = f
	}
	if cmd.Flags().Changed("out") {
		f, _ := cmd.Flags().GetString("out")
		opts.Out = f
	}
	if cmd.Flags().Changed("send-request") {
		f, _ := cmd.Flags().GetBool("send-request")
		opts.SendRequest = f
	}

	return opts
}

func ApplyRequestFlags(cmd *cobra.Command, req *httpcore.RequestConf) error {
	if cmd.Flags().Changed("method") {
		m, _ := cmd.Flags().GetString("method")
		req.SetMethod(strings.ToUpper(m))
	}

	if cmd.Flags().Changed("header") {
		h, _ := cmd.Flags().GetStringArray("header")
		headers, err := ParseKeyValues(h)
		if err != nil {
			return fmt.Errorf("invalid header: %w", err)
		}
		for k, v := range headers {
			req.AddHeader(k, v...)
		}
	}

	if cmd.Flags().Changed("query") {
		q, _ := cmd.Flags().GetStringArray("query")
		query, err := ParseKeyValues(q)
		if err != nil {
			return fmt.Errorf("invalid query parameter: %w", err)
		}
		for k, v := range query {
			req.AddQueryParam(k, v...)
		}
	}

	if cmd.Flags().Changed("cookie") {
		c, _ := cmd.Flags().GetStringArray("cookie")
		cookies, err := ParseKeySingleValue(c)
		if err != nil {
			return fmt.Errorf("invalid cookie: %w", err)
		}
		for k, v := range cookies {
			req.AddCookie(k, v)
		}
	}

	return nil
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

func ParseAttachments(cmd *cobra.Command) ([]byte, string, error) {
	if cmd.Flags().Changed("data") {
		return ParseData(cmd)
	}
	if cmd.Flags().Changed("form") {
		return ParseForm(cmd)
	}
	if cmd.Flags().Changed("part") {
		return ParseMultipart(cmd)
	}
	return nil, "", nil
}

func ParseData(cmd *cobra.Command) ([]byte, string, error) {
	arg, _ := cmd.Flags().GetString("data")
	arg = strings.TrimSpace(arg)

	buf := shared.BytesBuf()
	var ct *mimetype.MIME
	defer shared.PutBytesBuf(buf)

	if strings.HasPrefix(arg, "@") {
		path := strings.TrimPrefix(arg, "@")

		f, err := os.Open(path)
		if err != nil {
			return nil, "", fmt.Errorf("error opening file: %w", err)
		}
		defer f.Close()

		_, err = io.Copy(buf, f)
		if err != nil {
			return nil, "", fmt.Errorf("error reading content: %w", err)
		}

		ct = mimetype.Detect(buf.Bytes())
	} else {
		buf.WriteString(arg)

		ct = mimetype.Detect(buf.Bytes())

	}

	return buf.Bytes(), ct.String(), nil
}

func ParseForm(cmd *cobra.Command) (buf []byte, ct string, err error) {
	args, _ := cmd.Flags().GetStringArray("form")
	form := make(map[string][]string)

	for _, arg := range args {
		arg = strings.TrimSpace(arg)

		// FIXME: probably shouldn't use cut
		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, "", fmt.Errorf("wrong form syntax: must be exactly one '=' separator")
		}

		if strings.HasPrefix(val, "@") {
			buf := shared.BytesBuf()
			defer shared.PutBytesBuf(buf)

			path := strings.TrimPrefix(val, "@")
			f, err := os.Open(path)
			if err != nil {
				return nil, "", fmt.Errorf("error opening file: %w", err)
			}
			defer f.Close()

			_, err = io.Copy(buf, f)
			if err != nil {
				return nil, "", fmt.Errorf("error reading content: %w", err)
			}

			val = buf.String()
		}

		form[key] = append(form[key], val)
	}

	buf = httpcore.FormBytes(form)
	return buf, "application/x-www-form-urlencoded", nil
}

func ParseMultipart(cmd *cobra.Command) (buf []byte, ct string, err error) {
	args, _ := cmd.Flags().GetStringArray("part")

	builder := httpcore.NewMultipartBuilder()

	for _, arg := range args {
		arg = strings.TrimSpace(arg)

		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, "", fmt.Errorf("wrong part syntax: must be exactly one '=' separator")
		}

		if strings.HasPrefix(val, "@") {
			buf := shared.BytesBuf()
			defer shared.PutBytesBuf(buf)

			path := strings.TrimPrefix(val, "@")
			var f *os.File
			f, err = os.Open(path)
			if err != nil {
				return nil, "", fmt.Errorf("opening file: %w", err)
			}
			defer f.Close()

			err = builder.AddFileReader(key, path, f)
			if err != nil {
				return nil, "", fmt.Errorf("adding form file: %w", err)
			}

		} else if strings.HasPrefix(val, "<@") {
			buf := shared.BytesBuf()
			defer shared.PutBytesBuf(buf)

			path := strings.TrimPrefix(val, "<@")
			var f *os.File
			f, err = os.Open(path)
			if err != nil {
				return nil, "", fmt.Errorf("opening file: %w", err)
			}
			defer f.Close()

			_, err = io.Copy(buf, f)
			if err != nil {
				return nil, "", fmt.Errorf("reading file content: %w", err)
			}

			err = builder.AddTextField(key, buf.String())
			if err != nil {
				return nil, "", fmt.Errorf("adding text field: %w", err)
			}

		} else {
			err = builder.AddTextField(key, val)
			if err != nil {
				return nil, "", fmt.Errorf("adding text field: %w", err)
			}
		}
	}

	if buf, err = builder.Build(); err != nil {
		return nil, "", fmt.Errorf("building multipart: %w", err)
	}
	return buf, "multipart/form-data; boundary=" + builder.Boundary(), nil
}

func HasAttachments(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("data") ||
		cmd.Flags().Changed("form") ||
		cmd.Flags().Changed("part")
}
