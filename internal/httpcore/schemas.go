package httpcore

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func NewHttpRequest(urlArg, method string) (*HttpRequest, error) {
	req := HttpRequest{
		Method:      method,
		URL:         urlArg,
		QueryParams: make(map[string][]string),
		Headers:     make(map[string][]string),

		ShouldDumpRequest:   false,
		ShouldDumpResponse:  false,
		ShouldSendRequest:   true,
		ShouldSanitizeQuery: true,
	}

	q, err := ExtractQueryParams(urlArg)
	if err != nil {
		return nil, err
	}
	for k, v := range q {
		req.AddQueryParam(k, v...)
	}

	return &req, nil
}

func NewHttpRequestFromJSON(j []byte) (*HttpRequest, error) {
	var req HttpRequest
	r := bytes.NewReader(j)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

type HttpRequest struct {
	// serializable
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	Cookies     []Cookie            `json:"cookies"`

	// runtime opts
	ShouldDumpRequest     bool `json:"should_dump_request"`
	ShouldDumpResponse    bool `json:"should_dump_response"`
	ShouldSendRequest     bool `json:"should_send_request"`
	ShouldSanitizeQuery   bool `json:"should_sanitize_query"`
	ShouldSanitizeHeaders bool `json:"should_sanitize_headers"`
	ShouldSanitizeCookies bool `json:"should_sanitize_cookies"`

	// only through flags or methods
	body HttpBody
}

func (h HttpRequest) IsEmptyBody() bool {
	return h.body == nil
}

func (h HttpRequest) ToHTTP() (*http.Request, error) {
	req, err := http.NewRequest( // NOTE: shoud i use with context? and why?
		strings.ToUpper(h.Method),
		h.URL,
		h.body.Reader(),
	)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for key, val := range h.QueryParams {
		for _, v := range val {
			q.Add(key, v)
		}
	}
	req.URL.RawQuery = q.Encode()

	for k, vals := range h.Headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	for _, v := range h.Cookies {
		req.AddCookie(&http.Cookie{Name: v.Name, Value: v.Value})
	}

	if !h.IsEmptyBody() {
		req.Header.Add("Content-Type", h.body.ContentType())
	}

	return req, nil
}

// TODO: set, get, del, remove
func (h *HttpRequest) AddQueryParam(key string, val ...string) {
	if h.ShouldSanitizeQuery && len(val) == 0 {
		return
	}
	h.QueryParams[key] = append(h.QueryParams[key], val...)
}

// TODO: set, get, del, remove
func (h *HttpRequest) AddHeader(key string, val ...string) {
	if h.ShouldSanitizeHeaders && len(val) == 0 {
		return
	}
	h.Headers[key] = append(h.Headers[key], val...)
}

// TODO: set, get, del
// TODO: replace key, val with a whole cookie
func (h *HttpRequest) AddCookie(key string, val string) {
	if h.ShouldSanitizeCookies && len(val) == 0 {
		return
	}
	h.Cookies = append(h.Cookies, Cookie{Name: key, Value: val})
}

//func (h *HttpRequest) SetBodyHTML(b []byte) error {
//	c := http.DetectContentType(b)
//	if c != "text/html; charset=utf-8" {
//		return fmt.Errorf("not a valid HTML")
//	}
//	r := bytes.NewReader(b)
//	h.body = HttpBody{
//		ContentType: c,
//		Reader:      r,
//	}
//	return nil
//}
//
//func (h *HttpRequest) SetBodyForm(pairs map[string][]string) error {
//	formdata := url.Values{}
//	for k, val := range pairs {
//		for _, v := range val {
//			formdata.Add(k, v)
//		}
//	}
//	enc := formdata.Encode()
//	r := bytes.NewReader([]byte(enc))
//	h.body = HttpBody{
//		ContentType: "application/x-www-form-urlencoded",
//		Reader:      r,
//	}
//	return nil
//}
//
//func (h *HttpRequest) AddBodyMultipart() error {
//	return nil
//}

func (h *HttpRequest) SetBody(b HttpBody) {
	h.body = b
}

type HttpBody interface {
	ContentType() string
	Reader() io.Reader
	// TODO: Close() with cleanup if possible
}

type HttpBodyJSON []byte

func NewHttpBodyJSON(v any) (*HttpBodyJSON, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	res := HttpBodyJSON(buf.Bytes())
	return &res, nil
}

func (b HttpBodyJSON) ContentType() string {
	return "application/json; charset=utf-8"
}

func (b HttpBodyJSON) Reader() io.Reader {
	return bytes.NewReader(b)
}

func IsValidJSON(buf []byte) bool {
	trimmed := bytes.TrimSpace(buf)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if json.Valid(trimmed) {
			return true
		}
	}
	return false
}

type HttpBodyPlain string

func (h HttpBodyPlain) ContentType() string {
	return "text/plain; charset=utf-8"
}

func (h HttpBodyPlain) Reader() io.Reader {
	return strings.NewReader(string(h))
}

type HttpBodyHtml string

func (h HttpBodyHtml) ContentType() string {
	return "text/html; charset=utf-8"
}

func (h HttpBodyHtml) Reader() io.Reader {
	return strings.NewReader(string(h))
}

type HttpBodyForm map[string][]string

func (h *HttpBodyForm) Add(key, val string) {
	(*h)[key] = append((*h)[key], val)
}

func (h HttpBodyForm) ContentType() string {
	return "application/x-www-form-urlencoded"
}

func (h HttpBodyForm) Reader() io.Reader {
	q := url.Values(h)
	enc := q.Encode()
	return strings.NewReader(enc)
}

type HttpBodyMultipart struct {
	Boundary string
	Buf      *bytes.Buffer
	Mw       *multipart.Writer
}

func NewHttpBodyMultipart() *HttpBodyMultipart {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)

	return &HttpBodyMultipart{
		Boundary: mw.Boundary(),
		Buf:      buf,
		Mw: mw,
	}
}

func (h *HttpBodyMultipart) AddField(key, val string) error {
	return h.Mw.WriteField(key, val)
}

func (h *HttpBodyMultipart) AddFile(key, val string, file io.Reader) error {
	part, err := h.Mw.CreateFormFile(key, val)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	return err
}

func (h HttpBodyMultipart) ContentType() string {
	return "multipart/form-data; boundary="+h.Boundary 
}

func (h HttpBodyMultipart) Reader() io.Reader {
	return h.Buf
}

type CookieJar map[string]Cookie

func (j CookieJar) Get(d string) *Cookie {
	c := j[d]
	return &c
}

func (j *CookieJar) Load(r io.Reader) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(j)
}

func (j CookieJar) Save(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(j)
}

type Cookie struct {
	Name        string        `json:"name"`
	Value       string        `json:"value"`
	Domain      string        `json:"domain,omitzero"`
	Expires     time.Time     `json:"expires,omitzero"`
	HttpOnly    bool          `json:"http_only,omitzero"`
	MaxAge      int           `json:"max_age,omitzero"`
	Partitioned bool          `json:"partitioned,omitzero"`
	Path        string        `json:"path,omitzero"`
	SameSite    http.SameSite `json:"same_site,omitzero"` // TODO: replace with my own for proper serialization
	Secure      bool          `json:"secure,omitzero"`
}

// NOTE: the next whole thing is used only in test mode
func NewHttpResponse(r *http.Response) (HttpResponse, error) {
	resp := HttpResponse{
		Code:    uint(r.StatusCode),
		Headers: r.Header,
		// TODO: response cookies
	}
	if r.Body != nil {
		resp.body = r.Body
	}
	return resp, nil
}

type HttpResponse struct {
	Code    uint                `json:"code"`
	Headers map[string][]string `json:"headers"`
	Cookies []Cookie            `json:"cookies"`

	body io.Reader
}

func (h HttpResponse) IsSuccessful() bool {
	return h.Code >= http.StatusOK && h.Code < http.StatusMultipleChoices
}
