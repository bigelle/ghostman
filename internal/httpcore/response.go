package httpcore

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/lipgloss/tree"
)

func NewResponse(r *http.Response) (*Response, error) {
	resp := Response{
		Code:    r.StatusCode,
		Headers: r.Header,
	}

	if r.Body != nil {
		buf := &bytes.Buffer{}

		_, err := io.Copy(buf, r.Body)
		if err != nil {
			buf.Reset()
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		resp.body = buf
	}

	if cookies := r.Cookies(); cookies != nil {
		arr := make([]Cookie, len(cookies))
		for i, c := range cookies {
			v := Cookie{
				Name:        c.Name,
				Value:       c.Value,
				Domain:      c.Domain,
				Expires:     CookieTime{c.Expires},
				HttpOnly:    c.HttpOnly,
				MaxAge:      &c.MaxAge,
				Partitioned: c.Partitioned,
				Path:        c.Path,
				Secure:      c.Secure,
				SameSite:    SameSite(c.SameSite),
			}
			arr[i] = v
		}
		resp.Cookies = arr
	}

	return &resp, nil
}

type Response struct {
	Code    int                 `json:"code"`
	Headers map[string][]string `json:"headers"`
	Cookies []Cookie            `json:"cookies"`

	body *bytes.Buffer
}

func (r Response) String() string {
	t := tree.Root(fmt.Sprintf("%d %s", r.Code, http.StatusText(r.Code)))

	if len(r.Headers) > 0 {
		h := tree.Root("Headers:")
		for key, vals := range r.Headers {
			h.Child(fmt.Sprintf("%s: %s", key, strings.Join(vals, "; ")))
		}
		t.Child(h)
	}

	if len(r.Cookies) > 0 {
		c := tree.Root("Set-Cookie:")
		for _, cookie := range r.Cookies {
			c.Child(cookie.String())
		}
		t.Child(c)
	}

	if r.body != nil {
		b, err := func() (*tree.Tree, error) {
			size := FormatBytes(int64(r.body.Len()))

			ct, ok := r.Headers["Content-Type"]
			if !ok {
				return nil, fmt.Errorf("no content type, weird")
			}
			ctStr := strings.Join(ct, "; ")
			return tree.Root(fmt.Sprintf("Body: %s of %s", size, ctStr)), nil
		}()
		if err != nil {
			fmt.Println(err)
		}

		t.Child(b)
	}

	result := t.String()
	return result
}

func (h Response) IsSuccessful() bool {
	return h.Code >= http.StatusOK && h.Code < http.StatusMultipleChoices
}

//temporarily just a string
func (r Response) Body() string {
	return r.body.String()
}

