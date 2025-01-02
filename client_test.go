package jsonrpc2_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
	"github.com/macrat/go-jsonrpc2"
)

// Start a dummy server for testing.
//
// Returns:
// - conn: a connection to the server.
// - requests: a channel to receive requests from the client.
// - stop: a function to stop the server.
func StartDummyServer(t *testing.T) (conn io.ReadWriteCloser, requests chan jsonrpc2.Request[[]int], stop func()) {
	cli, srv := BiDirectionalPipe(t)

	requests = make(chan jsonrpc2.Request[[]int], 1)

	go func(t *testing.T) {
		for {
			reqs := jsonrpc2.NewMessageListForTest[jsonrpc2.Request[[]int]]()
			if err := json.NewDecoder(srv).Decode(&reqs); errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				t.Errorf("failed to decode request: %s", err)
			}

			resps := make([]jsonrpc2.Response[int], 0, len(reqs.Messages))

			for _, req := range reqs.Messages {
				requests <- req
				if req.ID != nil {
					if req.Method == "error" {
						resps = append(resps, jsonrpc2.Response[int]{
							Jsonrpc: jsonrpc2.VersionValue,
							Error: &jsonrpc2.Error{
								Code:    jsonrpc2.InternalErrorCode,
								Message: "debug error",
							},
							ID: req.ID,
						})
					} else {
						var sum int
						for _, v := range req.Params {
							sum += v
						}

						resps = append(resps, jsonrpc2.Response[int]{
							Jsonrpc: jsonrpc2.VersionValue,
							Result:  sum,
							ID:      req.ID,
						})
					}
				}
			}

			if len(resps) > 0 {
				if reqs.IsBatch {
					json.NewEncoder(srv).Encode(resps)
				} else {
					json.NewEncoder(srv).Encode(resps[0])
				}
			}
		}
	}(t)

	return cli, requests, func() {
		cli.Close()
		srv.Close()
		close(requests)
	}
}

func TestClient_Call(t *testing.T) {
	t.Parallel()

	cli, ch, stop := StartDummyServer(t)
	defer stop()

	client := jsonrpc2.NewClient(cli)
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var result int
	if err := client.Call(ctx, "call", []int{1, 2, 3}, &result); err != nil {
		t.Errorf("failed to call method: %s", err)
	}
	if result != 6 {
		t.Errorf("unexpected result: %d", result)
	}

	request := <-ch
	if request.Method != "call" {
		t.Errorf("unexpected method: %q", request.Method)
	}
	if diff := cmp.Diff([]int{1, 2, 3}, request.Params); diff != "" {
		t.Errorf("unexpected params: %s", diff)
	}
}

func TestClient_Notify(t *testing.T) {
	t.Parallel()

	cli, ch, stop := StartDummyServer(t)
	defer stop()

	client := jsonrpc2.NewClient(cli)
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := client.Notify(ctx, "notify", []int{1, 2, 3}); err != nil {
		t.Errorf("failed to call method: %s", err)
	}

	request := <-ch
	if request.Method != "notify" {
		t.Errorf("unexpected method: %q", request.Method)
	}
	if diff := cmp.Diff([]int{1, 2, 3}, request.Params); diff != "" {
		t.Errorf("unexpected params: %s", diff)
	}
	if request.ID != nil {
		t.Errorf("unexpected request id: %v", request.ID)
	}
}

func TestClient_Batch(t *testing.T) {
	t.Parallel()

	cli, ch, stop := StartDummyServer(t)
	defer stop()

	go func() {
		for x := range ch {
			t.Logf("     %v", x)
		}
	}()

	client := jsonrpc2.NewClient(cli)
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := []jsonrpc2.BatchRequest{
		{
			Method:   "batch1",
			Params:   []int{1, 2, 3},
			IsNotify: false,
		},
		{
			Method:   "batch2",
			Params:   []int{4, 5, 6},
			IsNotify: false,
		},
		{
			Method:   "batch3",
			Params:   []int{7, 8, 9},
			IsNotify: true,
		},
		{
			Method:   "error",
			Params:   []int{1, 2, 3},
			IsNotify: false,
		},
	}
	res, err := client.Batch(ctx, req)
	if err != nil {
		t.Fatalf("failed to invoke batch: %s", err)
	}

	expected := []*jsonrpc2.BatchResponse{
		{
			Method: "batch1",
			Params: []int{1, 2, 3},
			Result: json.RawMessage("6"),
			Error:  nil,
		},
		{
			Method: "batch2",
			Params: []int{4, 5, 6},
			Result: json.RawMessage("15"),
			Error:  nil,
		},
		nil,
		{
			Method: "error",
			Params: []int{1, 2, 3},
			Result: nil,
			Error: &jsonrpc2.Error{
				Code:    jsonrpc2.InternalErrorCode,
				Message: "debug error",
			},
		},
	}
	if diff := cmp.Diff(expected, res); diff != "" {
		t.Errorf("unexpected response:\n%s", diff)
	}
}
