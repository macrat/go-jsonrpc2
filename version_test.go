package jsonrpc2_test

import (
	"encoding/json"
	"testing"

	"github.com/macrat/go-jsonrpc2"
)

func TestVersion(t *testing.T) {
	tests := []struct {
		Input  string
		Error  string
		Output jsonrpc2.Version
	}{
		{
			Input:  `"2.0"`,
			Output: "2.0",
		},
		{
			Input: `"1.0"`,
			Error: `Invalid version: "jsonrpc" must be exactly "2.0" but got "1.0"`,
		},
		{
			Input: `2`,
			Error: `Invalid version: "jsonrpc" must be exactly "2.0" but got 2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			var version jsonrpc2.Version

			err := json.Unmarshal([]byte(tt.Input), &version)
			if (err != nil && err.Error() != tt.Error) || (err == nil && tt.Error != "") {
				t.Fatalf("unexpected error during unmarshal: %s", err)
			}
			if tt.Error != "" {
				return
			}

			if version != tt.Output {
				t.Errorf("unexpected value: want=%q got=%q", tt.Output, version)
			}
		})
	}
}
