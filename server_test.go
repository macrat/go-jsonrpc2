package jsonrpc2_test

import (
	"bufio"
	"context"
	"encoding/json"
	"testing"

	"github.com/macrat/go-jsonrpc2"
)

func BenchmarkServer(b *testing.B) {
	server := jsonrpc2.NewServer()

	server.On("add", jsonrpc2.Call(func(ctx context.Context, params []int) (int, error) {
		sum := 0
		for _, n := range params {
			sum += n
		}
		return sum, nil
	}))

	req, err := json.Marshal(jsonrpc2.Request[[]int]{
		Jsonrpc: "2.0",
		Method:  "add",
		Params:  []int{1, 2},
		ID:      jsonrpc2.Int64ID(123),
	})
	if err != nil {
		b.Fatalf("failed to prepare request: %s", err)
	}

	cli, srv := BiDirectionalPipe(nil)

	go server.ServeForOne(srv)

	reader := bufio.NewReader(cli)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cli.Write(req)
		reader.ReadLine()
	}
}
