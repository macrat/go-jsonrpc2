package jsonrpc2

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/goccy/go-json"
)

// Client is a JSON-RPC 2.0 client.
type Client struct {
	mu     sync.Mutex
	rw     io.ReadWriter
	ch     map[int64]chan<- Response[json.RawMessage]
	closer func()
	nextID int64
}

// NewClient creates a new JSON-RPC 2.0 client.
//
// The `rw` parameter is the read-writer to communicate with the server.
//
// This function starts a goroutine to read responses from the server.
// Please make sure to call `Close` to stop the goroutine when you are done.
func NewClient(rw io.ReadWriter) *Client {
	if rw == nil {
		panic("jsonrpc2: rw for jsonrpc2.NewClient is nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		rw:     rw,
		ch:     make(map[int64]chan<- Response[json.RawMessage]),
		closer: cancel,
	}

	go client.run(ctx)

	return client
}

func (c *Client) onResponse(r Response[json.RawMessage]) {
	if r.ID == nil {
		return
	}
	id := r.ID.i64
	if id == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ch, ok := c.ch[*id]
	if ok {
		delete(c.ch, *id)
		ch <- r
		close(ch)
	}
}

func (c *Client) run(ctx context.Context) {
	dec := json.NewDecoder(c.rw)

	for {
		var res messageList[Response[json.RawMessage]]

		err := dec.DecodeContext(ctx, &res)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			continue
		}

		for _, res := range res.Messages {
			c.onResponse(res)
		}
	}
}

// Close stops the client.
//
// A client cannot be used after it is closed.
func (c *Client) Close() error {
	c.closer()

	c.mu.Lock()
	defer c.mu.Unlock()

	for id, ch := range c.ch {
		delete(c.ch, id)
		close(ch)
	}

	return nil
}

func (c *Client) makeChan() (id int64, ch chan Response[json.RawMessage]) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id = c.nextID
	c.nextID++
	ch = make(chan Response[json.RawMessage], 1)
	c.ch[id] = ch

	return
}

// Call calls a method on the server.
//
// The response from the server is unmarshaled into the `result` parameter.
// If you do not need the response, use `Notify` instead.
func (c *Client) Call(ctx context.Context, name string, params any, result any) error {
	id, ch := c.makeChan()

	req := Request[any]{
		Jsonrpc: VersionValue,
		Method:  name,
		Params:  &params,
		ID:      Int64ID(id),
	}

	if _, err := req.WriteTo(c.rw); err != nil {
		c.mu.Lock()
		defer c.mu.Unlock()

		delete(c.ch, id)
		close(ch)

		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-ch:
			if res.Error != nil {
				return res.Error
			}

			return json.Unmarshal(res.Result, result)
		}
	}
}

// Notify sends a notification to the server.
//
// Even if the server replies something, the client will not receive it.
// If you need the response, use `Call` instead.
func (c *Client) Notify(ctx context.Context, name string, params any) error {
	req := Request[any]{
		Jsonrpc: VersionValue,
		Method:  name,
		Params:  &params,
	}

	_, err := req.WriteTo(c.rw)
	return err
}

// BatchRequest is a request for `Client.Batch`.
type BatchRequest struct {
	// Method is the method name to call.
	Method string

	// Params is the parameters to pass to the method.
	Params any

	// IsNotify is true if the request is a notification.
	// If true, the client will not receive a response from the server.
	IsNotify bool
}

// BatchResponse is a response from `Client.Batch`.
type BatchResponse struct {
	// Method is the method name that was called.
	Method string

	// Params is the parameters that were passed to the method.
	Params any

	// Result is the result of the method call.
	// Please use json.Unmarshal to read it.
	Result json.RawMessage

	// Error is the error that reported by the server.
	Error *Error
}

// Batch sends multiple requests to the server at once.
func (c *Client) Batch(ctx context.Context, reqs []BatchRequest) ([]*BatchResponse, error) {
	req := messageList[Request[any]]{
		IsBatch:  true,
		Messages: make([]Request[any], len(reqs)),
	}

	var ids []int64
	var chs []chan Response[json.RawMessage]

	c.mu.Lock()
	for i, r := range reqs {
		var id *ID
		if r.IsNotify {
			chs = append(chs, nil)
		} else {
			id = Int64ID(c.nextID)
			ids = append(ids, *id.i64)
			c.nextID++
			ch := make(chan Response[json.RawMessage], 1)
			chs = append(chs, ch)
			c.ch[*id.i64] = ch
		}

		req.Messages[i] = Request[any]{
			Jsonrpc: VersionValue,
			Method:  r.Method,
			Params:  &r.Params,
			ID:      id,
		}
	}
	c.mu.Unlock()

	destroy := func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		for _, id := range ids {
			if ch, ok := c.ch[id]; ok {
				delete(c.ch, id)
				close(ch)
			}
		}
	}

	if _, err := req.WriteTo(c.rw); err != nil {
		destroy()
		return nil, err
	}

	var resps []*BatchResponse

	for i, ch := range chs {
		if ch == nil {
			resps = append(resps, nil)
			continue
		}
		select {
		case <-ctx.Done():
			destroy()
			return nil, ctx.Err()
		case res := <-ch:
			resps = append(resps, &BatchResponse{
				Method: reqs[i].Method,
				Params: reqs[i].Params,
				Result: res.Result,
				Error:  res.Error,
			})
		}
	}

	return resps, nil
}
