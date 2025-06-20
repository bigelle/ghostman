package httpcore

import (
	"fmt"
	"io"
	"net/url"
	"os"

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

func (h BodySpec) Parse() (*Body, error) {
	switch h.Type {
	case "form":
		if h.FormData == nil {
			return nil, fmt.Errorf("empty form")
		}
		b := BodyForm(*h.FormData)
		return &b, nil
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

func (h BodySpec) toGeneric() (*Body, error) {
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

	b := buf.Bytes()
	ct := mimetype.Detect(b)

	return &Body{ContentType: ct.String(), Data: b}, nil
}

func (h BodySpec) toMultipart() (*Body, error) {
	if len(*h.MultipartFields) == 0 {
		return nil, fmt.Errorf("no multipart fields")
	}

	body := BodyMultipart(*h.MultipartFields)

	return &body, nil
}

type MultipartField struct {
	Name string  `json:"name"`
	Text *string `json:"text,omitempty"`
	File *string `json:"file,omitempty"`
}

type Body struct {
	ContentType string
	Data []byte
}

func BodyGeneric(b []byte) Body {
	ct := mimetype.Detect(b)
	return Body{ContentType: ct.String(), Data: b}
}

func BodyForm(form map[string][]string) Body {
	q := url.Values(form)
	return Body{
		ContentType: "application/x-www-url-formencoded",
		Data: []byte(q.Encode()),
	}
}

func BodyMultipart(parts []MultipartField) Body {
	return Body{}
}
