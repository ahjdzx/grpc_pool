# gRPC pool

The simple gRPC connection pool.

## Usage

```go
package main

import (
	"log"
	"time"

	"github.com/ahjdzx/grpc_pool"
)

func main() {
	pool := grpc_pool.NewPool(5, 30*time.Second)
	defer pool.Close()

	addr := "grpc_server:8080"
	conn, err := pool.GetConn(addr)
	if err != nil {
		log.Println(err)
		return
	}

	// do something with conn

	pool.Release(addr, conn, err)
}
```
