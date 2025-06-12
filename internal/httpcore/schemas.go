package httpcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

func NewRequest(urlArg, method string) (*Request, error) {
	req := Request{
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

func NewRequestFromJSON(j []byte) (*Request, error) {
	var req Request
	r := bytes.NewReader(j)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return nil, err
	}
	if req.Body != nil {
		body, err := req.Body.ToBody()
		if err != nil {
			return nil, err
		}
		req.SetBody(body)
	}
	return &req, nil
}

type Request struct {
	// serializable
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	Cookies     []Cookie            `json:"cookies"`
	Body        *BodySpec       `json:"body,omitempty"`

	// runtime opts
	ShouldDumpRequest     bool `json:"should_dump_request"`
	ShouldDumpResponse    bool `json:"should_dump_response"`
	ShouldSendRequest     bool `json:"should_send_request"`
	ShouldSanitizeQuery   bool `json:"should_sanitize_query"`
	ShouldSanitizeHeaders bool `json:"should_sanitize_headers"`
	ShouldSanitizeCookies bool `json:"should_sanitize_cookies"`

	// only through flags or methods
	body Body
}

func (h Request) IsEmptyBody() bool {
	return h.body == nil
}

func (h Request) GetBody() io.Reader {
	if h.IsEmptyBody() {
		return nil
	}
	return h.body.Reader()
}

func (h Request) ToHTTP() (*http.Request, error) {
	req, err := http.NewRequest( // NOTE: shoud i use with context? and why?
		strings.ToUpper(h.Method),
		h.URL,
		h.GetBody(),
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
func (h *Request) AddQueryParam(key string, val ...string) {
	if h.ShouldSanitizeQuery && len(val) == 0 {
		return
	}
	h.QueryParams[key] = append(h.QueryParams[key], val...)
}

// TODO: set, get, del, remove
func (h *Request) AddHeader(key string, val ...string) {
	if h.ShouldSanitizeHeaders && len(val) == 0 {
		return
	}
	h.Headers[key] = append(h.Headers[key], val...)
}

// TODO: set, get, del
// TODO: replace key, val with a whole cookie
func (h *Request) AddCookie(key string, val string) {
	if h.ShouldSanitizeCookies && len(val) == 0 {
		return
	}
	h.Cookies = append(h.Cookies, Cookie{Name: key, Value: val})
}

func (h *Request) SetBody(b Body) {
	h.body = b
}

type BodySpec struct {
	Type            string               `json:"type"`
	Text            *string              `json:"text,omitempty"`
	File            *string              `json:"file,omitempty"`
	FormData        *map[string][]string `json:"form_data,omitempty"`
	MultipartFields *[]MultipartField     `json:"multipart_fields"`
}

func (h BodySpec) ToBody() (Body, error) {
	switch h.Type {
	case "form":
		if h.FormData == nil {
			return nil, fmt.Errorf("empty form")
		}
		return BodyForm(*h.FormData), nil
	case "multipart":
		if h.MultipartFields == nil {
			return nil, fmt.Errorf("no multipart fields")
		}
		return h.toMultipart()
	case "content":
		if h.Text == nil {
			return nil, fmt.Errorf("no content")
		}
		return h.toGeneric()
	default:
		return nil, fmt.Errorf("unknown body type: %s", h.Type)
	}
}

func (h BodySpec) toGeneric() (*BodyGeneric, error) {
	buf := make([]byte, 1024)
	if h.File != nil {
		f, err := os.Open(*h.File)
		if err != nil {
			return nil, err
		}
		n, err := f.Read(buf)
		if err != nil {
			return nil, err
		}
		buf = buf[:n]
	} else if h.Text != nil {
		r := strings.NewReader(*h.Text)
		n, err := r.Read(buf)
		if err != nil {
			return nil, err
		}
		buf = buf[:n]
	}
	ct := mimetype.Detect(buf)
	return NewBodyGeneric(ct.String(), buf), nil
}

func (h BodySpec) toMultipart() (*BodyMultipart, error) {
	body := NewBodyMultipart()

	for _, field := range *h.MultipartFields {
		if field.Text != nil {
			if err := body.AddField(field.Name, *field.Text); err != nil {
				return nil, err
			}
		}
		if field.File != nil {
			b, err := os.ReadFile(*field.File)
			if err != nil {
				return nil, err
			}
			if err := body.AddFile(field.Name, *field.File, b); err != nil {
				return nil, err
			}
		}
	}
	return body, nil
}

type MultipartField struct {
	Name string  `json:"name"`
	Text *string `json:"text,omitempty"`
	File *string `json:"file,omitempty"`
}

type Body interface {
	ContentType() string
	Reader() io.Reader
	// TODO: Close() with cleanup if possible
}

type BodyGeneric struct {
	ct string
	r  io.Reader
}

func NewBodyGeneric(ct string, b []byte) *BodyGeneric {
	buf := bytes.NewReader(b)
	return &BodyGeneric{ct: ct, r: buf}
}

func (h BodyGeneric) ContentType() string {
	return h.ct
}

func (h BodyGeneric) Reader() io.Reader {
	return h.r
}

type BodyForm map[string][]string

func (h *BodyForm) Add(key, val string) {
	if h == nil {
		return // not panicing
	}
	(*h)[key] = append((*h)[key], val)
}

func (h BodyForm) ContentType() string {
	return "application/x-www-url-formencoded"
}

func (h BodyForm) Reader() io.Reader {
	q := url.Values{}
	for k, vals := range h {
		for _, v := range vals {
			q.Add(k, v)
		}
	}
	return strings.NewReader(q.Encode())
}

type BodyMultipart struct {
	Boundary string
	Mw       *multipart.Writer
	Buf      *bytes.Buffer
}

func NewBodyMultipart() *BodyMultipart {
	buf := bytes.NewBuffer([]byte{})
	mw := multipart.NewWriter(buf)
	return &BodyMultipart{
		Boundary: mw.Boundary(),
		Mw:       mw,
		Buf:      buf,
	}
}

func (h *BodyMultipart) AddField(key, val string) error {
	return h.Mw.WriteField(key, val)
}

func (h *BodyMultipart) AddFile(key, val string, file []byte) error {
	r := bytes.NewReader(file)
	header := textproto.MIMEHeader{
		"Content-Disposition": []string{
			"form-data", fmt.Sprintf("name=\"%s\"", key), fmt.Sprintf("filename=\"%s\"", val),
		},
		"Content-Type": []string{
			mimetype.Detect(file).String(),
		},
	}
	part, err := h.Mw.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, r)
	return err
}

func (h BodyMultipart) ContentType() string {
	return "multipart/form-data; boundary=" + h.Boundary
}

func (h BodyMultipart) Reader() io.Reader {
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
