package httpcore

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func (h HttpRequest) Body() io.Reader {
	if h.IsEmptyBody() {
		return nil
	}
	return h.body.Reader()
}

func (h HttpRequest) ToHTTP() (*http.Request, error) {
	req, err := http.NewRequest( // NOTE: shoud i use with context? and why?
		strings.ToUpper(h.Method),
		h.URL,
		h.Body(),
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

func (h *HttpRequest) SetBody(b HttpBody) {
	h.body = b
}

type HttpBody interface {
	ContentType() string
	Reader() io.Reader
	// TODO: Close() with cleanup if possible
}

func genericReader(b []byte) io.Reader {
	return bytes.NewReader(b)
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
	return genericReader(b)
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
	return genericReader([]byte(h))
}

type HttpBodyHtml string

func (h HttpBodyHtml) ContentType() string {
	return "text/html; charset=utf-8"
}

func (h HttpBodyHtml) Reader() io.Reader {
	return genericReader([]byte(h))
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
		Mw:       mw,
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
	return "multipart/form-data; boundary=" + h.Boundary
}

func (h HttpBodyMultipart) Reader() io.Reader {
	return h.Buf
}

type HttpBodyOctetStream []byte

func (h HttpBodyOctetStream) ContentType() string {
	return "application/octet-stream"
}

func (h HttpBodyOctetStream) Reader() io.Reader {
	return genericReader(h)
}

type HttpBodyXML []byte

func (h HttpBodyXML) ContentType() string {
	return "application/xml; charset=utf-8"
}

func (h HttpBodyXML) Reader() io.Reader {
	return genericReader(h)
}

type HttpBodyCSS []byte

func (h HttpBodyCSS) ContentType() string {
	return "text/css; charset=utf-8"
}

func (h HttpBodyCSS) Reader() io.Reader {
	return genericReader(h)
}

type HttpBodyJavascript []byte

func (h HttpBodyJavascript) ContentType() string {
	return "text/javascript; charset=utf-8"
}

func (h HttpBodyJavascript) Reader() io.Reader {
	return genericReader(h)
}

type HttpBodyImage struct {
	Ct string
	B  []byte
}

func NewHttpBodyImage(b []byte) (*HttpBodyImage, error) {
	ct := http.DetectContentType(b)
	if !strings.HasPrefix(ct, "image") {
		return nil, fmt.Errorf("not an image")
	}
	return &HttpBodyImage{
		Ct: ct,
		B:  b,
	}, nil
}

func (h HttpBodyImage) ContentType() string {
	return h.Ct
}

func (h HttpBodyImage) Reader() io.Reader {
	return genericReader(h.B)
}

type HttpBodyAudio struct {
	Ct string
	B    []byte
}

func NewHttpBodyAudio(b []byte) (*HttpBodyAudio, error) {
	ct := http.DetectContentType(b)
	if !strings.HasPrefix(ct, "audio") {
		return nil, fmt.Errorf("not an audio")
	}
	return &HttpBodyAudio{
		Ct: ct,
		B:  b,
	}, nil
}

func (h HttpBodyAudio) ContentType() string {
	return h.Ct
}

func (h HttpBodyAudio) Reader() io.Reader {
	return genericReader(h.B)
}

type HttpBodyVideo struct {
	Ct string
	B    []byte
}

func NewHttpBodyVideo(b []byte) (*HttpBodyVideo, error) {
	ct := http.DetectContentType(b)
	if !strings.HasPrefix(ct, "video") {
		return nil, fmt.Errorf("not a video")
	}
	return &HttpBodyVideo{
		Ct: ct,
		B:  b,
	}, nil
}

func (h HttpBodyVideo) ContentType() string {
	return h.Ct
}

func (h HttpBodyVideo) Reader() io.Reader {
	return genericReader(h.B)
}

type HttpBodyPDF []byte

func (h HttpBodyPDF) ContentType() string {
	return "application/pdf"
}

func (h HttpBodyPDF) Reader() io.Reader {
	return genericReader(h)
}

type HttpBodyZip []byte

func (h HttpBodyZip) ContentType() string {
	return "application/zip"
}

func (h HttpBodyZip) Reader() io.Reader {
	return genericReader(h)
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
