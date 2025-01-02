package jsonrpc2

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestServer(t *testing.T) {
	server := NewServer()

	server.On("add", Call(func(ctx context.Context, params []int) (int, error) {
		sum := 0
		for _, n := range params {
			sum += n
		}
		return sum, nil
	}))

	var message string
	server.On("notify", Notify(func(ctx context.Context, s string) error {
		message += s
		return nil
	}))

	ptr := func(v any) *any { return &v }

	tests := []struct {
		Request  RawRequest
		Response *Response[*any]
		Message  string
	}{
		{
			RawRequest{
				Jsonrpc: "2.0",
				Method:  "add",
				Params:  json.RawMessage("[1,2]"),
				ID:      Int64ID(123),
			},
			&Response[*any]{
				Jsonrpc: "2.0",
				Result:  ptr(3),
				ID:      Int64ID(123),
			},
			"",
		},
		{
			RawRequest{
				Jsonrpc: "2.0",
				Method:  "add",
				Params:  json.RawMessage("[-1,1]"),
				ID:      Int64ID(123),
			},
			&Response[*any]{
				Jsonrpc: "2.0",
				Result:  ptr(0),
				ID:      Int64ID(123),
			},
			"",
		},
		{
			RawRequest{
				Jsonrpc: "2.0",
				Method:  "notify",
				Params:  json.RawMessage(`"hello"`),
			},
			nil,
			"hello",
		},
		{
			RawRequest{
				Jsonrpc: "2.0",
				Method:  "notify",
				Params:  json.RawMessage(`"world"`),
				ID:      Int64ID(123),
			},
			&Response[*any]{
				Jsonrpc: "2.0",
				Result:  ptr(nil),
				ID:      Int64ID(123),
			},
			"world",
		},
		{
			RawRequest{
				Jsonrpc: "2.0",
				Method:  "noSuchMethod",
				Params:  json.RawMessage(`null`),
				ID:      Int64ID(123),
			},
			&Response[*any]{
				Jsonrpc: "2.0",
				Error:   &ErrMethodNotFound,
				ID:      Int64ID(123),
			},
			"",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i, tt := range tests {
		res := server.call(ctx, tt.Request)

		if diff := cmp.Diff(tt.Response, res, cmp.AllowUnexported(ID{})); diff != "" {
			t.Errorf("test[%d]: unexpected response:\n%s", i, diff)
		}

		if message != tt.Message {
			t.Errorf("test[%d]: unexpected message: %s", i, message)
		}
		message = ""
	}
}

func BenchmarkServer_Call_success(b *testing.B) {
	server := NewServer()

	server.On("add", Call(func(ctx context.Context, params []int) (int, error) {
		sum := 0
		for _, n := range params {
			sum += n
		}
		return sum, nil
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.call(ctx, RawRequest{
			Jsonrpc: "2.0",
			Method:  "add",
			Params:  json.RawMessage("[1,2]"),
			ID:      Int64ID(123),
		})
	}
}

func BenchmarkServer_Call_error(b *testing.B) {
	server := NewServer()

	server.On("add", Call(func(ctx context.Context, params []int) (int, error) {
		return 0, ErrInternalError
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.call(ctx, RawRequest{
			Jsonrpc: "2.0",
			Method:  "add",
			Params:  json.RawMessage("[1,2]"),
			ID:      Int64ID(123),
		})
	}
}
