package jsonrpc2

import (
	"fmt"
	"io"

	"github.com/goccy/go-json"
)

type countWriter struct {
	out   io.Writer
	count int64
}

func (w *countWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	w.count += int64(n)
	return n, err
}

// Request represents a JSON-RPC 2.0 request.
//
// You usually do not need to use this struct directly.
// Please use `Client.Call` or `Client.Notify` method.
type Request[T any] struct {
	Jsonrpc Version `json:"jsonrpc"`
	Method  string  `json:"method"`
	Params  T       `json:"params,omitempty"`
	ID      *ID     `json:"id,omitempty"`
}

// RawRequest is a variant of `Request` that uses `json.RawMessage` for `Params`.
//
// This type is used for handling requests in `Server`.
type RawRequest = Request[json.RawMessage]

// NewRequest creates a new JSON-RPC 2.0 request.
func NewRequest[T any](id *ID, method string, params T) *Request[T] {
	return &Request[T]{
		Jsonrpc: VersionValue,
		Method:  method,
		Params:  params,
		ID:      id,
	}
}

// WriteTo writes JSON-RPC 2.0 request to `io.Writer`.
func (r *Request[T]) WriteTo(w io.Writer) (int64, error) {
	cw := &countWriter{out: w}
	if err := json.NewEncoder(cw).Encode(r); err != nil {
		return cw.count, err
	}

	return cw.count, nil
}

// Response represents a JSON-RPC 2.0 response.
//
// You usually do not need to use this struct directly.
type Response[T any] struct {
	Jsonrpc Version `json:"jsonrpc"`
	Result  T       `json:"result,omitempty"`
	Error   *Error  `json:"error,omitempty"`
	ID      *ID     `json:"id"`
}

// NewSuccessResponse creates a new JSON-RPC 2.0 success response.
func NewSuccessResponse[T any](id *ID, result T) *Response[T] {
	return &Response[T]{
		Jsonrpc: VersionValue,
		Result:  result,
		ID:      id,
	}
}

// NewErrorResponse creates a new JSON-RPC 2.0 error response.
func NewErrorResponse(id *ID, err Error) *Response[any] {
	return &Response[any]{
		Jsonrpc: VersionValue,
		Error:   &err,
		ID:      id,
	}
}

// WriteTo writes JSON-RPC 2.0 response to `io.Writer`.
func (r *Response[T]) WriteTo(w io.Writer) (int64, error) {
	cw := &countWriter{out: w}
	if err := json.NewEncoder(cw).Encode(r); err != nil {
		return cw.count, err
	}

	return cw.count, nil
}

// Error represents a JSON-RPC 2.0 error object.
//
// This struct can be used as `error` value.
// The server uses this value if handlers return it as an error.
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Data    any       `json:"data,omitempty"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}

type ErrorCode int64

const (
	ParseErrorCode     ErrorCode = -32700
	InvalidRequestCode ErrorCode = -32600
	MethodNotFoundCode ErrorCode = -32601
	InvalidParamsCode  ErrorCode = -32602
	InternalErrorCode  ErrorCode = -32603
)

var (
	ErrParseError     = Error{Code: ParseErrorCode, Message: "Parse error"}
	ErrInvalidRequest = Error{Code: InvalidRequestCode, Message: "Invalid Request"}
	ErrMethodNotFound = Error{Code: MethodNotFoundCode, Message: "Method not found"}
	ErrInvalidParams  = Error{Code: InvalidParamsCode, Message: "Invalid params"}
	ErrInternalError  = Error{Code: InternalErrorCode, Message: "Internal error"}
)

func (e ErrorCode) String() string {
	switch e {
	case ParseErrorCode:
		return fmt.Sprintf("Parse error (%d)", ParseErrorCode)
	case InvalidRequestCode:
		return fmt.Sprintf("Invalid Request (%d)", InvalidRequestCode)
	case MethodNotFoundCode:
		return fmt.Sprintf("Method not found (%d)", MethodNotFoundCode)
	case InvalidParamsCode:
		return fmt.Sprintf("Invalid params (%d)", InvalidParamsCode)
	case InternalErrorCode:
		return fmt.Sprintf("Internal error (%d)", InternalErrorCode)
	}

	if -32000 <= e && e <= -32099 {
		return fmt.Sprintf("Server error (%d)", e)
	}

	return fmt.Sprintf("Error (%d)", e)
}

// messageList is a helper type for marshaling/unmarshaling JSON-RPC 2.0 messages.
// This type supports both single message and batch message.
type messageList[T any] struct {
	IsBatch  bool
	Messages []T
}

func (l messageList[T]) MarshalJSON() ([]byte, error) {
	if len(l.Messages) == 1 && l.IsBatch {
		return json.Marshal(l.Messages[0])
	}

	return json.Marshal(l.Messages)
}

func (l *messageList[T]) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		l.IsBatch = true
		return json.Unmarshal(data, &l.Messages)
	}

	var m T
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	l.Messages = []T{m}

	return nil
}

func (l *messageList[T]) WriteTo(w io.Writer) (int64, error) {
	cw := &countWriter{out: w}
	if err := json.NewEncoder(cw).Encode(l); err != nil {
		return cw.count, err
	}

	return cw.count, nil
}
