package httpcmd_test

import (
	"net/http"
	"testing"

	"github.com/bigelle/ghostman/cli/httpcmd"
)

func Test_HttpRequest_Request(t *testing.T){
	httpreq := httpcmd.HttpRequest{
		Method: http.MethodGet,
		URL: "https://catfact.ninja/fact",
		QueryParams: map[string][]string{
			"max_length": {"42"},
		},
	}
	req, err := httpreq.Request()
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
