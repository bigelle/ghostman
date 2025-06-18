package httpcore

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"strings"

	"github.com/bigelle/ghostman/internal/shared"
	"github.com/gabriel-vasile/mimetype"
)

type BodySpec struct {
	Type            string               `json:"type"`
	Text            *string              `json:"text,omitempty"`
	File            *string              `json:"file,omitempty"`
	FormData        *map[string][]string `json:"form_data,omitempty"`
	MultipartFields *[]MultipartField    `json:"multipart_fields"`
}

func (h BodySpec) Parse() (Body, error) {
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
	if h.File == nil && h.Text == nil {
		return nil, fmt.Errorf("no text or file specified")
	}

	buf := shared.BytesBuf()
	defer shared.PutBytesBuf(buf)

	if h.File != nil {
		f, err := os.Open(*h.File)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		_, err = io.Copy(buf, f)
		if err != nil {
			return nil, err
		}
	} else if h.Text != nil {
		_, err := buf.WriteString(*h.Text)
		if err != nil {
			return nil, err
		}
	}
	b := buf.Bytes()
	ct := mimetype.Detect(b)
	return &BodyGeneric{Ct: ct.String(), B: b}, nil
}

func (h BodySpec) toMultipart() (*BodyMultipart, error) {
	if len(*h.MultipartFields) == 0 {
		return nil, fmt.Errorf("no multipart fields")
	}

	body := NewBodyMultipart()

	for _, field := range *h.MultipartFields {
		if field.Text != nil {
			if err := body.AddField(field.Name, *field.Text); err != nil {
				return nil, err
			}
		}
		if field.File != nil {
			buf := shared.BytesBuf()
			defer shared.PutBytesBuf(buf)

			f, err := os.Open(*field.File)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			_, err = io.Copy(buf, f)
			if err != nil {
				return nil, err
			}

			if err := body.AddFile(field.Name, *field.File, buf.Bytes()); err != nil {
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
	Len() int64
	Close() error
}

type BodyGeneric struct {
	Ct string
	B  []byte
}

func (h BodyGeneric) ContentType() string {
	return h.Ct
}

func (h BodyGeneric) Reader() io.Reader {
	return bytes.NewReader(h.B)
}

func (h BodyGeneric) Len() int64 {
	return int64(len(h.B))
}

func (b BodyGeneric) Close() error {
	return nil
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
	q := url.Values(h)
	return strings.NewReader(q.Encode())
}

func (b BodyForm) Len() int64 {
	// FIXME:
	return 0
}

func (b BodyForm) Close() error {
	return nil
}

type BodyMultipart struct {
	Boundary string
	Mw       *multipart.Writer
	Buf      *bytes.Buffer
}

func NewBodyMultipart() *BodyMultipart {
	buf := &bytes.Buffer{}
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

func (b BodyMultipart) Len() int64 {
	return int64(b.Buf.Len())
}

func (b BodyMultipart) Close() error {
	return b.Mw.Close()
}
