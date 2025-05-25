package httpcmd

import (
	"reflect"
	"testing"
)

func TestUtils_parseHTTPHeaders(t *testing.T) {
	cases := []struct {
		Name      string
		Input     []string
		Output    map[string][]string
		ExpectErr bool
	}{
		{
			Name:  "one header",
			Input: []string{"Accept:application/json,text/plain"},
			Output: map[string][]string{
				"Accept": {"application/json", "text/plain"},
			},
			ExpectErr: false,
		},
		{
			Name: "multiple headers",
			Input: []string{
				"Accept:application/json,text/plain",
				"Content-Type:application/json",
			},
			Output: map[string][]string{
				"Accept": {"application/json", "text/plain"},
				"Content-Type": {"application/json"},
			},
			ExpectErr: false,
		},
		{
			Name: "wrong format",
			Input: []string{
				"Accept:application/json,text/plain",
				"Content-Type:application:json", // 2 : signs
			},
			ExpectErr: true,
		},
		{
			Name: "extra spaces",
			Input: []string{
				"Accept: application/json, text/plain",
				"Content-Type: application/json",
			},
			Output: map[string][]string{
				"Accept": {"application/json", "text/plain"},
				"Content-Type": {"application/json"},
			},
			ExpectErr: false,
		},
	}

	for _, tcase := range cases {
		t.Run(tcase.Name, func(t *testing.T) {
			result, err := parseHTTPKeyValues(tcase.Input)
			if err != nil && !tcase.ExpectErr {
				t.Errorf("unexpected error: %s", err.Error())
			} else {
				return
			}

			if !reflect.DeepEqual(*result, tcase.Output) {
				t.Errorf("test result differs from expected result: \n expected %+v\n got %+v\n", tcase.Output, result)
			}
		})
	}
}
