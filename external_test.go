package jsonrpc2_test

import (
	"context"
	"io"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/go-jsonrpc2"
)

type ReadWriteCloser struct {
	r       io.ReadCloser
	w       io.WriteCloser
	onWrite func([]byte)
}

func (rw *ReadWriteCloser) Read(p []byte) (int, error) {
	return rw.r.Read(p)
}

func (rw *ReadWriteCloser) Write(p []byte) (int, error) {
	if rw.onWrite != nil {
		rw.onWrite(p)
	}
	return rw.w.Write(p)
}

func (rw *ReadWriteCloser) Close() error {
	rw.r.Close()
	rw.w.Close()
	return nil
}

func BiDirectionalPipe(t *testing.T) (client *ReadWriteCloser, server *ReadWriteCloser) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	if t == nil {
		client = &ReadWriteCloser{r2, w1, nil}
		server = &ReadWriteCloser{r1, w2, nil}
		return
	} else {
		client = &ReadWriteCloser{r1, w2, func(p []byte) {
			t.Logf("--> %s", string(p))
		}}
		server = &ReadWriteCloser{r2, w1, func(p []byte) {
			t.Logf("<-- %s", string(p))
		}}
		return
	}
}

func Test_clientAndServer(t *testing.T) {
	t.Parallel()

	cli, srv := BiDirectionalPipe(t)

	var notificationCount uint64

	type SubParams struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	type SubResult struct {
		Value int `json:"value"`
	}

	func() {
		server := jsonrpc2.NewServer()

		server.On("add", jsonrpc2.Call(func(ctx context.Context, params []int) (int, error) {
			sum := 0
			for _, n := range params {
				sum += n
			}
			return sum, nil
		}))

		server.On("sub", jsonrpc2.Call(func(ctx context.Context, params SubParams) (SubResult, error) {
			return SubResult{params.A - params.B}, nil
		}))

		server.On("count", jsonrpc2.Notify(func(ctx context.Context, params int) error {
			atomic.AddUint64(&notificationCount, uint64(params))
			return nil
		}))

		go server.ServeForOne(srv)
	}()

	func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client := jsonrpc2.NewClient(cli)
		defer client.Close()

		if err := client.Notify(ctx, "count", 10); err != nil {
			t.Fatalf("failed to call count: %s", err)
		}
		if err := client.Notify(ctx, "count", 3); err != nil {
			t.Fatalf("failed to call count: %s", err)
		}
		var countRes any
		if err := client.Call(ctx, "count", 2, &countRes); err != nil {
			t.Fatalf("failed to call count: %s", err)
		}
		if countRes != nil {
			t.Errorf("unexpected result of count: %v", countRes)
		}

		var sum int
		if err := client.Call(ctx, "add", []int{1, 2, 3}, &sum); err != nil {
			t.Fatalf("failed to call add: %s", err)
		}
		if sum != 6 {
			t.Errorf("unexpected result of add: %d", sum)
		}

		var sub SubResult
		if err := client.Call(ctx, "sub", SubParams{A: 10, B: 3}, &sub); err != nil {
			t.Fatalf("failed to call sub: %s", err)
		}
		if diff := cmp.Diff(SubResult{7}, sub); diff != "" {
			t.Errorf("unexpected result of sub:\n%s", diff)
		}

		var notFound any
		if err := client.Call(ctx, "notFound", nil, &notFound); err == nil {
			t.Errorf("unexpected success of notFound: %v", notFound)
		} else if err.Error() != "Method not found (-32601)" {
			t.Errorf("unexpected error of notFound: %s", err)
		}

		if diff := cmp.Diff(uint64(15), atomic.LoadUint64(&notificationCount)); diff != "" {
			t.Errorf("unexpected notification count: %s", diff)
		}

		batchReq := []jsonrpc2.BatchRequest{
			{Method: "add", Params: []int{1, 2, 3}},
			{Method: "sub", Params: SubParams{A: 10, B: 3}},
		}
		batchRes, err := client.Batch(ctx, batchReq)
		if err != nil {
			t.Fatalf("failed to call batch: %s", err)
		}
		batchExpected := []*jsonrpc2.BatchResponse{
			{Method: "add", Params: []int{1, 2, 3}, Result: []byte("6")},
			{Method: "sub", Params: SubParams{A: 10, B: 3}, Result: []byte(`{"value":7}`)},
		}
		if diff := cmp.Diff(batchExpected, batchRes); diff != "" {
			t.Errorf("unexpected result of batch:\n%s", diff)
		}
	}()
}
