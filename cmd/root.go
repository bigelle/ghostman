package cmd

import (
	"os"
	"strings"

	"github.com/bigelle/ghostman/internal/httpcore"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:     "ghostman",
	Short:   "deez nuts",
	Args:    cobra.ExactArgs(1),
	PreRunE: PreRun,
	RunE:    Run,
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().Bool(
		"from-file",
		false,
		"set an HTTP method used for http/s request",
	)
	RootCmd.PersistentFlags().StringP(
		"method",
		"M",
		"GET",
		"set an HTTP method used for http/s request",
	)

	RootCmd.PersistentFlags().Bool("dump-request", false, "dump the whole request")
	RootCmd.PersistentFlags().Bool("dump-response", false, "dump the whole response")
	RootCmd.PersistentFlags().Bool("send-request", true, "send request")
	RootCmd.PersistentFlags().Bool("sanitize-cookies", true, "omits empty or malformed cookies")
	RootCmd.PersistentFlags().Bool("sanitize-headers", true, "omits empty or malformed headers")
	RootCmd.PersistentFlags().Bool("sanitize-query", true, "omits empty or malformed query parameters")

	RootCmd.PersistentFlags().StringArrayP(
		"query",
		"Q",
		[]string{},
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)
	RootCmd.PersistentFlags().StringArrayP(
		"cookie",
		"C",
		[]string{},
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)
	RootCmd.PersistentFlags().StringArrayP(
		"header",
		"H",
		[]string{},
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)

	RootCmd.PersistentFlags().String(
		"data",
		"",
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)
	RootCmd.PersistentFlags().StringArray(
		"form",
		[]string{},
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)
	RootCmd.PersistentFlags().StringArray(
		"part",
		[]string{},
		"sets Content-Type header to 'text/html' and adds passed string as a body",
	)
}

func PreRun(cmd *cobra.Command, args []string) error {
	arg := args[0]
	isFromFile, _ := cmd.Flags().GetBool("from-file")
	if isFromFile {
		// FIXME: TEMPORARILY just trying to read it as a HttpRequest
		return PreRunHttpFile(cmd, args)
	}
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		return PreRunHttp(cmd, args)
	}
	return nil
}

func Run(cmd *cobra.Command, args []string) error {
	if req, ok := cmd.Context().Value(ctxKeyHttpReq).(*httpcore.HttpRequest); ok {
		return RunHttp(req)
	}
	return nil
}
