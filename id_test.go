package jsonrpc2_test

import (
	"encoding/json"
	"testing"

	"github.com/macrat/go-jsonrpc2"
)

func TestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input string
		Error string
		Raw   any
		Str   string
	}{
		{
			Input: `123`,
			Raw:   int64(123),
			Str:   "123",
		},
		{
			Input: `"hello"`,
			Raw:   string("hello"),
			Str:   `"hello"`,
		},
		{
			Input: `null`,
			Raw:   nil,
			Str:   `null`,
		},
		{
			Input: `"null"`,
			Raw:   string("null"),
			Str:   `"null"`,
		},
		{
			Input: `123.456`,
			Error: `Invalid ID: ID have to be either integer, string, or null but got "123.456"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			var id jsonrpc2.ID

			err := json.Unmarshal([]byte(tt.Input), &id)
			if (err != nil && err.Error() != tt.Error) || (err == nil && tt.Error != "") {
				t.Fatalf("unexpected error during unmarshal: %s", err)
			}
			if tt.Error != "" {
				return
			}

			if id.Raw() != tt.Raw {
				t.Errorf("unexpected raw value: want=%#v got=%#v", tt.Raw, id.Raw())
			}

			if id.String() != tt.Str {
				t.Errorf("unexpected string value: want=%#v got=%#v", tt.Str, id.String())
			}

			j, err := json.Marshal(id)
			if err != nil {
				t.Fatalf("unexpected error during marshal: %s", err)
			}

			if string(j) != tt.Input {
				t.Errorf("unexpected marshalled value: want=%#v got=%#v", tt.Input, string(j))
			}
		})
	}
}

func BenchmarkID_UnmarshalJSON(b *testing.B) {
	var id jsonrpc2.ID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = id.UnmarshalJSON([]byte("123"))
		_ = id.UnmarshalJSON([]byte(`"hello"`))
	}
}

func BenchmarkID_MarshalJSON(b *testing.B) {
	i64 := jsonrpc2.Int64ID(123)
	str := jsonrpc2.StringID("hello")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = i64.MarshalJSON()
		_, _ = str.MarshalJSON()
	}
}

func BenchmarkID_String(b *testing.B) {
	i64 := jsonrpc2.Int64ID(123)
	str := jsonrpc2.StringID("hello")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = i64.String()
		_ = str.String()
	}
}
