package httpcore

import (
	"encoding/json"
	"fmt"
	"io"
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
	Name        string     `json:"name"`
	Value       string     `json:"value"`
	Domain      string     `json:"domain,omitzero"`
	Expires     CookieTime `json:"expires,omitzero"`
	HttpOnly    bool       `json:"http_only,omitzero"`
	MaxAge      int        `json:"max_age,omitzero"`
	Partitioned bool       `json:"partitioned,omitzero"`
	Path        string     `json:"path,omitzero"`
	SameSite    SameSite   `json:"same_site,omitzero"`
	Secure      bool       `json:"secure,omitzero"`
}

type CookieTime struct {
	time.Time
}

func (c CookieTime) MarshalJSON() ([]byte, error) {
	if c.IsZero() {
		return []byte("null"), nil
	}
	return fmt.Appendf(nil, `"%s"`, c.In(time.FixedZone("GMT", 0)).Format(time.RFC1123)), nil
}

func (c *CookieTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.Time = time.Time{}
		return nil
	}

	str := string(data[1 : len(data)-1])

	t, err := time.Parse(time.RFC1123, str)
	if err != nil {
		return err
	}

	c.Time = t.In(time.FixedZone("GMT", 0))
	return nil
}

type SameSite int

const (
	SameSiteDefaultMode SameSite = iota + 1
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

func (s SameSite) String() string {
	return [...]string{"", "Lax", "Strict", "None"}[s-1]
}

func (s SameSite) FromString(ss string) SameSite {
	return map[string]SameSite{
		"":       SameSiteDefaultMode,
		"Lax":    SameSiteLaxMode,
		"Strict": SameSiteStrictMode,
		"None":   SameSiteNoneMode,
	}[ss]
}

func (s SameSite) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SameSite) UnmarshalJSON(data []byte) error {
	var i string
	err := json.Unmarshal(data, &i)
	if err != nil {
		return nil
	}
	*s = s.FromString(i)
	return nil
}
