package httpcore

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/lipgloss/tree"
	"github.com/gabriel-vasile/mimetype"
)

type Response struct {
	resp *http.Response
	body []byte // used for reading after closing the resp.Body
}

func (r Response) ToString() (str string, err error) {
	dumpBody := false
	if r.body != nil {
		rdr := bytes.NewReader(r.body)
		r.resp.Body = io.NopCloser(rdr)
		dumpBody = true
	}

	var dump []byte
	dump, err = DumpResponse(r.resp, dumpBody)
	if err != nil {
		return "", fmt.Errorf("dumping response safely: %w", err)
	}

	parts := strings.Split(string(dump), "\r\n")
	if len(parts) == 1 {
		return "", fmt.Errorf("malformed response")
	}

	parts = parts[1:]

	var headers []string
	var cookies []string
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			break
		}
		if strings.HasPrefix(part, "Set-Cookie:") {
			cookies = append(cookies, strings.TrimSpace(strings.TrimPrefix(part, "Set-Cookie:")))
			continue
		}
		headers = append(headers, strings.TrimSpace(part))
	}

	t := tree.Root(Status(r.resp.StatusCode))

	if len(headers) != 0 {
		h := tree.Root("Headers:")
		for _, header := range headers {
			h.Child(header)
		}

		t.Child(h)
	}

	if len(cookies) != 0 {
		c := tree.Root("Set-Cookie:")
		for _, cookie := range cookies {
			c.Child(cookie)
		}

		t.Child(c)
	}

	bytesIndex := bytes.Index(dump, []byte("\r\n\r\n"))
	if bytesIndex != -1 && bytesIndex+4 != len(dump) {
		body := dump[bytesIndex+4:]

		size := FormatBytes(int64(len(body)))

		ct := r.resp.Header.Get("Content-Type")
		if ct == "" {
			mimeCt := mimetype.Detect(body)
			ct = mimeCt.String()
		}

		t.Child(fmt.Sprintf("Body: %s of %s", size, ct))
	}

	return t.String(), nil
}

func (r *Response) WriteBodyTo(w io.Writer) error {
	n, err := w.Write(r.body)
	if err != nil {
		return fmt.Errorf("writing body to the other destination: %w", err)
	}
	if n != len(r.body) {
		return fmt.Errorf("wrote %d bytes, expected %d", n, len(r.body))
	}
	return nil
}
