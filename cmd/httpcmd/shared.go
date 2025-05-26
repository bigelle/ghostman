package httpcmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httputil"

	"github.com/bigelle/ghostman/internal/httpcore"
	"github.com/spf13/cobra"
)

func parseCommand(cmd *cobra.Command, args []string) error {
	req, err := httpcore.NewHttpRequest(args[0], cmd.Name())
	if err != nil {
		return err
	}
	//TODO: parse other args

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, "httpReq", *req)
	cmd.SetContext(withVal)
	return nil
}

func dumpRequestSafely(req *http.Request) ([]byte, error) {
	clone, err := cloneRequest(req)
	if err != nil {
		return nil, err
	}
	return httputil.DumpRequestOut(clone, req.Body != nil)
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	clone := req.Clone(req.Context())
	if bodyBytes != nil {
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	return clone, nil
}

