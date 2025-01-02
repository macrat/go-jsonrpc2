# Go JSON-RPC 2.0

[![Go Reference](https://pkg.go.dev/badge/github.com/macrat/go-jsonrpc2.svg)](https://pkg.go.dev/github.com/macrat/go-jsonrpc2)

A super easy [JSON-RPC 2.0](https://www.jsonrpc.org/specification) client and server library for Go.

## Example
### Server

```go
package main

import (
	"context"
	"log"

	"github.com/macrat/go-jsonrpc2"
)

func main() {
	server := jsonrpc2.NewServer()

	// Add a method.
	server.On("sum", jsonrpc2.Call(func(ctx context.Context, xs []int) (int, error) {
		sum := 0
		for _, x := range xs {
			sum += x
		}
		return sum, nil
	}))

	// Add a notification handler.
	server.On("notify", jsonrpc2.Notify(func(ctx context.Context, message string) error {
		log.Printf("notify: %s", message)
		return nil
	}))

	l, err := jsonrpc2.NewTCPListener(":1234")
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(server.Serve(l))
}
```

### Client

```go
package main

import (
	"context"
	"net"

	"github.com/macrat/go-jsonrpc2"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:1234")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	client := jsonrpc2.NewClient(conn)

	// Call	`sum` method.
	var sum int
	err := client.Call(ctx, "sum", []int{1, 2, 3}, &sum)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sum: %d", sum)

	// Send `notify` notification.
	err = client.Notify(ctx, "notify", "Hello, World!")
	if err != nil {
		log.Fatal(err)
	}
}
```
