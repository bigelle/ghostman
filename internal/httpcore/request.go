package httpcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss/tree"
	"github.com/gabriel-vasile/mimetype"
)

func NewRequest(reqUrl string, body *Body) (*RequestConf, error) {
	_, err := url.ParseRequestURI(reqUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid request URL: %w", err)
	}

	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body.Data)
	}

	req, err := http.NewRequest(http.MethodGet, reqUrl, r)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if body != nil {
		req.Header.Add("Content-Type", body.ContentType)
	}

	return &RequestConf{req: req}, nil
}

func NewRequestFromJSON(j []byte) (req *RequestConf, err error) {
	ser := RequestSerializable{
		Method:      http.MethodGet,
		QueryParams: make(map[string][]string),
		Headers:     make(map[string][]string),
	}

	r := bytes.NewReader(j)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	if err = dec.Decode(&ser); err != nil {
		return nil, fmt.Errorf("error reading request config: %w", err)
	}

	var body *Body
	if ser.Body != nil {
		body, err = ser.Body.Parse()
		if err != nil {
			return nil, fmt.Errorf("error parsing body: %w", err)
		}
	} else {
		body = &Body{}
	}

	request, err := http.NewRequest(ser.Method, ser.URL, bytes.NewReader(body.Data))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := request.URL.Query()
	if ser.QueryParams != nil {
		for key, val := range ser.QueryParams {
			for _, v := range val {
				q.Add(key, v)
			}
		}
	}
	request.URL.RawQuery = q.Encode()

	if ser.Headers != nil {
		for k, vals := range ser.Headers {
			for _, v := range vals {
				request.Header.Add(k, v)
			}
		}
	}

	if ser.Cookies != nil {
		for _, v := range ser.Cookies {
			request.AddCookie(&http.Cookie{Name: v.Name, Value: v.Value})
		}
	}

	if body != nil {
		request.Header.Add("Content-Type", body.ContentType)
	}

	return &RequestConf{req: request}, nil
}

type RequestConf struct {
	req *http.Request
}

func (r RequestConf) ToString() (string, error) {
	if r.req == nil {
		r.ToHTTP()
	}

	dump, err := DumpRequest(r.req)
	if err != nil {
		return "", err
	}

	rows := strings.Split(string(dump), "\r\n")
	if len(rows) == 0 {
		return "", fmt.Errorf("malformed request")
	}

	rows = rows[1:]

	var headers []string
	var cookies []string
	for _, row := range rows {
		if strings.TrimSpace(row) == "" {
			break
		}
		if strings.HasPrefix(row, "Cookie:") {
			// temporarily just print them without any formatting
			// for now just raw key=value
			cookies = append(cookies, strings.TrimSpace(strings.TrimPrefix(row, "Cookie:")))
			continue
		}
		headers = append(headers, row)
	}

	t := tree.Root(fmt.Sprintf("%s %s", Method(r.req.Method), r.req.URL))

	if len(headers) > 0 {
		h := tree.Root("Headers:")
		for _, header := range headers {
			h.Child(header)
		}
		t.Child(h)
	}

	if len(cookies) > 0 {
		c := tree.Root("Cookie:")
		for _, cookie := range cookies {
			c.Child(cookie)
		}
		t.Child(c)
	}

	bodyIndex := bytes.Index(dump, []byte("\r\n\r\n"))
	if bodyIndex != -1 && bodyIndex+4 != len(dump) {
		body := dump[bodyIndex+4:]
		size := FormatBytes(int64(len(body)))
		ct := r.req.Header.Get("Content-Type")
		if ct == "" {
			mimeCt := mimetype.Detect(body)
			ct = mimeCt.String()
		}
		t.Child(fmt.Sprintf("Body: %s of %s", size, ct))
	}

	return t.String(), nil
}

func (c RequestConf) SetMethod(m string) {
	c.req.Method = m
}

func (h RequestConf) GetBody() io.ReadCloser {
	return h.req.Body
}

func (r *RequestConf) ToHTTP() *http.Request {
	return r.req
}

// TODO: set, get, del, remove
func (r *RequestConf) AddQueryParam(key string, vals ...string) {
	q := r.req.URL.Query()
	for _, val := range vals {
		q.Add(key, val)
	}
	r.req.URL.RawQuery = q.Encode()
}

// TODO: set, get, del, remove
func (c *RequestConf) AddHeader(key string, vals ...string) {
	for _, val := range vals {
		c.req.Header.Add(key, val)
	}
}

// TODO: set, get, del
// TODO: replace key, val with a whole cookie
func (c *RequestConf) AddCookie(key string, val string) {
	c.req.AddCookie(&http.Cookie{Name: key, Value: val})
}

func (r *RequestConf) SetBody(b *Body) {
	r.req.Body = &BytesReadCloser{*bytes.NewReader(b.Data)}
	r.req.Header.Add("Content-Type", b.ContentType)
}

type RequestSerializable struct {
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	QueryParams map[string][]string `json:"query_params,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty"`
	Cookies     []Cookie            `json:"cookies,omitempty"`
	Body        *BodySpec           `json:"body,omitempty"`
}
