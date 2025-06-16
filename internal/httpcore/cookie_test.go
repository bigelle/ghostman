package httpcore

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCookie_Marshal(t *testing.T) {
	m := 42
	c := Cookie{
		Name:        "foo",
		Value:       "bar",
		Domain:      "example.com",
		Expires:     CookieTime{time.Date(2025, time.January, 1, 0, 0, 0, 0, time.FixedZone("GMT", 0))},
		HttpOnly:    true,
		MaxAge: &m,
		Partitioned: true,
		Path:        "/",
		SameSite:    SameSiteLaxMode,
		Secure:      true,
	}
	b, err := json.Marshal(c)
	require.NoError(t, err)

	var shouldBe Cookie
	err = json.Unmarshal(b, &shouldBe)
	require.NoError(t, err)
	
	assert.Equal(t, c, shouldBe)
}

