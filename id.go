package jsonrpc2

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/goccy/go-json"
)

var (
	ErrInvalidID     = errors.New("Invalid ID")
	ErrInvalidIDType = fmt.Errorf("%w: %w", ErrInvalidID, errors.New("ID have to be either integer, string, or null"))
)

// ID respesents ID in JSON-RPC 2.0 request/response.
type ID struct {
	i64 *int64
	str *string
}

// Int64ID creates a new ID with int64 value.
func Int64ID(i int64) *ID {
	return &ID{
		i64: &i,
	}
}

// StringID creates a new ID with string value.
func StringID(s string) *ID {
	return &ID{
		str: &s,
	}
}

// NullID creates a new ID with null value.
//
// This value is used for error response.
// Please use normal nil value for notification.
func NullID() *ID {
	return &ID{}
}

// Raw returns raw ID value in int64, string, or nil.
func (id ID) Raw() any {
	if id.i64 != nil {
		return *id.i64
	} else if id.str != nil {
		return *id.str
	} else {
		return nil
	}
}

// String returns string representation of ID.
func (id ID) String() string {
	if id.i64 != nil {
		return strconv.FormatInt(*id.i64, 10)
	} else if id.str != nil {
		return strconv.Quote(*id.str)
	} else {
		return "null"
	}
}

// MarshalJSON implements json.Marshaler interface.
func (id ID) MarshalJSON() ([]byte, error) {
	if id.i64 != nil {
		return json.Marshal(id.i64)
	} else if id.str != nil {
		return json.Marshal(id.str)
	} else {
		return []byte("null"), nil
	}
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (id *ID) UnmarshalJSON(data []byte) error {
	id.i64 = nil
	id.str = nil

	if bytes.Equal(data, []byte("null")) || len(data) == 0 {
		return nil
	}

	if data[0] == '-' || ('0' <= data[0] && data[0] <= '9') {
		if i, err := strconv.ParseInt(string(data), 10, 64); err == nil {
			id.i64 = &i
			return nil
		}
	}

	if data[0] == '"' {
		if err := json.Unmarshal(data, &id.str); err == nil {
			return nil
		}
	}

	return fmt.Errorf("%w but got %q", ErrInvalidIDType, string(data))
}
