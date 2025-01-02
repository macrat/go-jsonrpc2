package jsonrpc2

import (
	"bytes"
	"errors"
	"fmt"
)

var (
	ErrInvalidVersion = errors.New(`Invalid version: "jsonrpc" must be exactly "2.0"`)
)

// Version represents "jsonrpc" value in JSON-RPC 2.0 request/response.
// This value is always "2.0".
type Version string

const VersionValue Version = "2.0"

func (v *Version) UnmarshalJSON(data []byte) error {
	if !bytes.Equal(data, []byte(`"2.0"`)) {
		return fmt.Errorf("%w but got %s", ErrInvalidVersion, string(data))
	}

	*v = VersionValue
	return nil
}
