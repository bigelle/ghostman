package httpcore

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

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

