package httpcore

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"sync"

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

func (h BodySpec) Parse() ([]byte, error) {
	switch h.Type {
	case "form":
		if h.FormData == nil {
			return nil, fmt.Errorf("empty form")
		}
		b := FormBytes(*h.FormData)
		return b, nil
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

func (h BodySpec) toGeneric() ([]byte, error) {
	if h.File == nil && h.Text == nil {
		return nil, fmt.Errorf("no text or file specified")
	}

	buf := shared.BytesBuf()
	defer shared.PutBytesBuf(buf)

	if h.File != nil {
		f, err := os.Open(*h.File)
		if err != nil {
			return nil, fmt.Errorf("error opening file: %w", err)
		}
		defer f.Close()

		_, err = io.Copy(buf, f)
		if err != nil {
			return nil, fmt.Errorf("error reading content: %w", err)
		}
	} else if h.Text != nil {
		buf.WriteString(*h.Text)
	}

	return buf.Bytes(), nil
}

func (h BodySpec) toMultipart() (buf []byte, err error) {
	if h.MultipartFields != nil && len(*h.MultipartFields) == 0 {
		return nil, fmt.Errorf("no multipart fields")
	}

	builder := NewMultipartBuilder()

	for _, part := range *h.MultipartFields {
		if part.Text != "" {
			err = builder.AddTextField(part.Name, part.Text)
			if err != nil {
				return nil, fmt.Errorf("writing text part: %w", err)
			}
		} else if part.File != "" {
			var f *os.File
			f, err = os.Open(part.File)
			if err != nil {
				return nil, fmt.Errorf("opening file for reading: %w", err)
			}
			err = builder.AddFileReader(part.Name, f.Name(), f)
			if err != nil {
				return nil, fmt.Errorf("adding file to multipart: %w", err)
			}
		}
	}

	buf, err = builder.Build()
	if err != nil {
		return nil, fmt.Errorf("closing multipart builder: %w", err)
	}

	return buf, nil
}

type MultipartField struct {
	Name string `json:"name"`
	Text string `json:"text,omitempty"`
	File string `json:"file,omitempty"`
}

type MultipartBuilder struct {
	buf      *bytes.Buffer
	mw       *multipart.Writer
	boundary string
	finished bool
	mu       sync.Mutex
}

func NewMultipartBuilder() *MultipartBuilder {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)

	return &MultipartBuilder{
		buf:      buf,
		mw:       mw,
		boundary: mw.Boundary(),
		finished: false,
		mu:       sync.Mutex{},
	}
}

func (mb *MultipartBuilder) AddTextField(field, value string) (err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.finished {
		return fmt.Errorf("builder is already closed")
	}

	err = mb.mw.WriteField(field, value)
	if err != nil {
		return fmt.Errorf("writing text field: %w", err)
	}

	return nil
}

func (mb *MultipartBuilder) AddFile(field, file string, content []byte) (err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.finished {
		return fmt.Errorf("builder is already closed")
	}

	ct := mimetype.Detect(content)
	header := textproto.MIMEHeader{
		"Content-Disposition": []string{
			"form-data",
			fmt.Sprintf(`name="%s"`, field),
			fmt.Sprintf(`filename="%s"`, file),
		},
		"Content-Type": []string{
			ct.String(),
		},
	}

	var part io.Writer
	part, err = mb.mw.CreatePart(header)
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}

	_, err = part.Write(content)
	if err != nil {
		return fmt.Errorf("writing to form file: %w", err)
	}

	return nil
}

// It IS going to drain your reader
func (mb *MultipartBuilder) AddFileReader(field, file string, r io.Reader) (err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.finished {
		return fmt.Errorf("builder is already closed")
	}

	buf, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading from reader: %w", err)
	}

	ct := mimetype.Detect(buf)

	header := textproto.MIMEHeader{
		"Content-Disposition": []string{
			"form-data",
			fmt.Sprintf(`name="%s"`, field),
			fmt.Sprintf(`filename="%s"`, file),
		},
		"Content-Type": []string{
			ct.String(),
		},
	}

	var part io.Writer
	part, err = mb.mw.CreatePart(header)
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}

	_, err = part.Write(buf)
	if err != nil {
		return fmt.Errorf("writing to form file: %w", err)
	}

	return nil
}

func (mb *MultipartBuilder) Build() (content []byte, err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.finished {
		return nil, fmt.Errorf("builder is already closed")
	}

	if err = mb.mw.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	mb.finished = true

	return mb.buf.Bytes(), nil
}

func (mb *MultipartBuilder) Boundary() string {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	return mb.boundary
}

func FormBytes(form map[string][]string) []byte {
	q := url.Values(form)
	return []byte(q.Encode())
}
