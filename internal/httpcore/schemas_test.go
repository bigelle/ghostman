package httpcore

import (
	"net/http"
	"testing"

)

func Test_HttpRequest_Request(t *testing.T) {
	httpreq := Request{
		Method: http.MethodGet,
		URL:    "https://catfact.ninja/fact",
		QueryParams: &map[string][]string{
			"max_length": {"42"},
		},
	}
	req, err := httpreq.ToHTTP()
	if err != nil {
		t.FailNow()
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		t.FailNow()
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	t.Log(resp.StatusCode)
}
