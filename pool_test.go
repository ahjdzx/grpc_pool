package grpc_pool

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

func TestGRPCPool(t *testing.T) {
	testPool(t, 0, 30*time.Second)
	testPool(t, 2, 30*time.Second)
}

type greeterServer struct{}

func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{
		Message: "Hello " + req.GetName(),
	}, nil
}

func testPool(t *testing.T, size int, ttl time.Duration) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	server := grpc.NewServer()
	pb.RegisterGreeterServer(server, &greeterServer{})

	go server.Serve(l)
	defer server.Stop()

	pool := NewPool(size, ttl)
	defer pool.Close()

	addr := l.Addr().String()

	for i := 0; i < 100; i++ {
		c, err := pool.GetConn(addr, grpc.WithInsecure())
		if err != nil {
			t.Fatal(err)
		}

		resp := pb.HelloReply{}

		err = c.Invoke(context.TODO(), "/helloworld.Greeter/SayHello", &pb.HelloRequest{Name: "Xin"}, &resp)
		if err != nil {
			t.Fatal(err)
		}

		if resp.Message != "Hello Xin" {
			t.Fatalf("Expect Hello Xin, but got %v", resp.Message)
		}

		pool.Release(addr, c, nil)

		pool.Lock()
		if count := len(pool.conns[addr]); count > size {
			pool.Unlock()
			t.Fatalf("pool size %d is greater than expected %d", count, size)
		}
		pool.Unlock()
	}
}
