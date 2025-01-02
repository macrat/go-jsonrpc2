package jsonrpc2

import (
	"context"
	"errors"
	"io"
	"sort"

	"github.com/goccy/go-json"
)

// Handler is a base interface for JSON-RPC 2.0 handlers.
//
// In almost all cases, you do not need to implement this interface directly.
// Please use jsonrpc2.Call or jsonrpc2.Notify to create a handler.
type Handler interface {
	ServeJSONRPC2(context.Context, RawRequest) (any, error)
}

// Call creates a new JSON-RPC 2.0 handler for a method that returns a result.
func Call[I, O any](f func(context.Context, I) (O, error)) Handler {
	return callHandler[I, O](f)
}

type callHandler[I, O any] func(context.Context, I) (O, error)

func (f callHandler[I, O]) ServeJSONRPC2(ctx context.Context, r RawRequest) (any, error) {
	var params I

	if err := json.Unmarshal(r.Params, &params); err != nil {
		return nil, ErrInvalidParams
	}

	return f(ctx, params)
}

// Notify creates a new JSON-RPC 2.0 handler for a method that does not return a result.
func Notify[I any](f func(context.Context, I) error) Handler {
	return notifyHandler[I](f)
}

type notifyHandler[I any] func(context.Context, I) error

func (f notifyHandler[I]) ServeJSONRPC2(ctx context.Context, r RawRequest) (any, error) {
	var params I

	if err := json.Unmarshal(r.Params, &params); err != nil {
		return nil, ErrInvalidParams
	}

	return nil, f(ctx, params)
}

type handlerInfo struct {
	name    string
	handler Handler
}

// Server is a JSON-RPC 2.0 server.
type Server struct {
	handlers           []handlerInfo
	maxConcurrentCalls int
	semaphore          chan struct{}
}

// NewServer creates a new JSON-RPC 2.0 server.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		maxConcurrentCalls: 100,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.semaphore = make(chan struct{}, s.maxConcurrentCalls)
	return s
}

// ServerOption is a type for server options.
type ServerOption func(*Server)

// WithMaxConcurrentCalls specifies the maximum number of concurrent calls the server can handle.
// If this option is not specified, the default value is 100.
//
// maxConcurrent must be greater than 0.
func WithMaxConcurrentCalls(maxConcurrent int) ServerOption {
	if maxConcurrent <= 0 {
		panic("maxConcurrent must be greater than 0")
	}
	return func(s *Server) {
		s.maxConcurrentCalls = maxConcurrent
	}
}

// ServeJSONRPC2 implements the Handler interface.
//
// Do not call this method directly.
func (s *Server) ServeJSONRPC2(ctx context.Context, r RawRequest) (any, error) {
	idx := sort.Search(len(s.handlers), func(i int) bool {
		return s.handlers[i].name >= r.Method
	})

	if idx >= len(s.handlers) || s.handlers[idx].name != r.Method {
		return nil, ErrMethodNotFound
	}

	return s.handlers[idx].handler.ServeJSONRPC2(ctx, r)
}

// On registers a new handler for a method.
//
// If the handler returns `Error` struct as an error, the server sends an error as-is to the client.
func (s *Server) On(name string, m Handler) {
	idx := sort.Search(len(s.handlers), func(i int) bool {
		return s.handlers[i].name >= name
	})

	if idx < len(s.handlers) && s.handlers[idx].name == name {
		s.handlers[idx].handler = m
	} else {
		s.handlers = append(s.handlers, handlerInfo{})
		copy(s.handlers[idx+1:], s.handlers[idx:])
		s.handlers[idx] = handlerInfo{name, m}
	}
}

// call invokes a single request and returns the response.
// The return type uses a pointer to any to make differentation between nil and zero values.
func (s *Server) call(ctx context.Context, r RawRequest) *Response[*any] {
	result, err := s.ServeJSONRPC2(ctx, r)
	if r.ID == nil {
		return nil
	}

	resp := Response[*any]{
		Jsonrpc: VersionValue,
		ID:      r.ID,
	}

	var errRes Error
	if errors.As(err, &errRes) {
		resp.Error = &errRes
	} else if err != nil {
		resp.Error = &Error{Code: InternalErrorCode, Message: "Internal error"}
	} else {
		resp.Result = &result
	}

	return &resp
}

func (s *Server) callAll(ctx context.Context, rw io.ReadWriter, rs messageList[RawRequest]) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if !rs.IsBatch {
		r := s.call(ctx, rs.Messages[0])
		if r != nil {
			r.WriteTo(rw)
		}
		return
	}

	ch := make(chan *Response[*any], len(rs.Messages))

	for _, req := range rs.Messages {
		s.semaphore <- struct{}{}

		go func(req RawRequest) {
			defer func() { <-s.semaphore }()

			ch <- s.call(ctx, req)
		}(req)
	}

	results := make([]Response[*any], 0, len(rs.Messages))
	i := 0
	for resp := range ch {
		if resp != nil {
			results = append(results, *resp)
		}
		i++
		if i == len(rs.Messages) {
			break
		}
	}

	close(ch)

	json.NewEncoder(rw).Encode(results)
}

// ServeForOne reads requests from the given io.ReadWriter and sends responses to it.
func (s *Server) ServeForOne(rw io.ReadWriter) {
	r := json.NewDecoder(rw)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		var rs messageList[RawRequest]
		if err := r.Decode(&rs); errors.Is(err, io.EOF) {
			return
		} else if err != nil {
			NewErrorResponse(NullID(), ErrInvalidRequest).WriteTo(rw)
			continue
		}

		s.callAll(ctx, rw, rs)
	}
}

// Serve accepts connections from the given listener and handles them.
func (s *Server) Serve(l Listener) error {
	for {
		rw, err := l.Accept()
		if err != nil {
			return err
		}

		go s.ServeForOne(rw)
	}
}
