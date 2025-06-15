package httpcore

import (
	"io"
	"net/http"
)

// NOTE: the next whole thing is used only in test mode
func NewResponse(r *http.Response) (Response, error) {
	resp := Response{
		Code:    uint(r.StatusCode),
		Headers: r.Header,
		// TODO: response cookies
	}
	if r.Body != nil {
		resp.body = r.Body
	}
	return resp, nil
}

type Response struct {
	Code    uint                `json:"code"`
	Headers map[string][]string `json:"headers"`
	Cookies []Cookie            `json:"cookies"`

	body io.Reader
}

func (h Response) IsSuccessful() bool {
	return h.Code >= http.StatusOK && h.Code < http.StatusMultipleChoices
}
