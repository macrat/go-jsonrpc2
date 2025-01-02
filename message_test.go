package jsonrpc2_test

import (
	"bytes"
	"testing"

	"github.com/macrat/go-jsonrpc2"
)

func BenchmarkRequest_WriteTo(b *testing.B) {
	var buf bytes.Buffer

	req := jsonrpc2.NewRequest(jsonrpc2.Int64ID(123), "test", "hello world")

	req.WriteTo(&buf)
	buf.Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.WriteTo(&buf)
		buf.Reset()
	}
}

func BenchmarkResponse_WriteTo(b *testing.B) {
	var buf bytes.Buffer

	res := jsonrpc2.NewSuccessResponse(jsonrpc2.Int64ID(123), "hello world")

	res.WriteTo(&buf)
	buf.Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.WriteTo(&buf)
		buf.Reset()
	}
}
