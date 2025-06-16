package httpcore

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bigelle/ghostman/internal/shared"
)

func NewResponse(r *http.Response) (*Response, error) {
	resp := Response{
		Code:    r.StatusCode,
		Headers: r.Header,
	}
	if r.Body != nil {
		resp.body = r.Body
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

	body io.Reader
}

func (r Response) String() string {
	buf := shared.StringBuilder()
	defer shared.PutStringBuilder(buf)

	fmt.Fprintf(buf, "%d %s\n", r.Code, http.StatusText(r.Code))

	for key, vals := range r.Headers {
		for _, val := range vals {
			fmt.Fprintf(buf, "  %s: %s\n", key, val)
		}
	}

	for _, val := range r.Cookies {
		fmt.Fprintf(buf, "  Set-Cookie: %s\n", val)
	}

	return buf.String()
}

func (h Response) IsSuccessful() bool {
	return h.Code >= http.StatusOK && h.Code < http.StatusMultipleChoices
}
