package grpc_pool

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

func TestGRPCPool(t *testing.T) {
	testPool(t, 0)
	testPool(t, 2)
}

type greeterServer struct {
	pb.UnsafeGreeterServer
}

func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{
		Message: "Hello " + req.GetName(),
	}, nil
}

func testPool(t *testing.T, size int) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	server := grpc.NewServer()
	pb.RegisterGreeterServer(server, &greeterServer{})

	go server.Serve(l)
	defer server.Stop()

	addr := l.Addr().String()

	dialFunc := func() (*grpc.ClientConn, error) {
		return grpc.Dial(addr, grpc.WithInsecure())
	}

	pool, err := NewPool(size, dialFunc)
	if err != nil {
		return
	}
	defer pool.Close()

	for i := 0; i < 100; i++ {
		c, err := pool.GetConn()
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

	}
}

func BenchmarkWithoutPool(b *testing.B) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		b.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	server := grpc.NewServer()
	pb.RegisterGreeterServer(server, &greeterServer{})

	go server.Serve(l)
	defer server.Stop()

	addr := l.Addr().String()
	dialFunc := func() (*grpc.ClientConn, error) {
		return grpc.Dial(addr, grpc.WithInsecure())
	}

	c, err := dialFunc()
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()

	resp := pb.HelloReply{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = c.Invoke(context.TODO(), "/helloworld.Greeter/SayHello", &pb.HelloRequest{Name: "Xin"}, &resp)
		if err != nil {
			b.Fatal(err)
		}
	}

	if resp.Message != "Hello Xin" {
		b.Fatalf("Expect Hello Xin, but got %v", resp.Message)
	}
}

func BenchmarkWithPool(b *testing.B) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		b.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	server := grpc.NewServer()
	pb.RegisterGreeterServer(server, &greeterServer{})

	go server.Serve(l)
	defer server.Stop()

	addr := l.Addr().String()
	dialFunc := func() (*grpc.ClientConn, error) {
		return grpc.Dial(addr, grpc.WithInsecure())
	}

	pool, err := NewPool(10, dialFunc)
	if err != nil {
		return
	}
	defer pool.Close()

	resp := pb.HelloReply{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c, err := pool.GetConn()
		if err != nil {
			b.Fatal(err)
		}

		err = c.Invoke(context.TODO(), "/helloworld.Greeter/SayHello", &pb.HelloRequest{Name: "Xin"}, &resp)
		if err != nil {
			b.Fatal(err)
		}
	}

	if resp.Message != "Hello Xin" {
		b.Fatalf("Expect Hello Xin, but got %v", resp.Message)
	}
}

func BenchmarkWithoutPoolParallel(b *testing.B) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		b.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	server := grpc.NewServer()
	pb.RegisterGreeterServer(server, &greeterServer{})

	go server.Serve(l)
	defer server.Stop()

	addr := l.Addr().String()
	dialFunc := func() (*grpc.ClientConn, error) {
		return grpc.Dial(addr, grpc.WithInsecure())
	}

	c, err := dialFunc()
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()

	b.ResetTimer()
	b.RunParallel(func(ppb *testing.PB) {
		for ppb.Next() {
			resp := pb.HelloReply{}
			err = c.Invoke(context.TODO(), "/helloworld.Greeter/SayHello", &pb.HelloRequest{Name: "Xin"}, &resp)
			if err != nil {
				b.Fatal(err)
			}
			if resp.Message != "Hello Xin" {
				b.Fatalf("Expect Hello Xin, but got %v", resp.Message)
			}
		}
	})
}

func BenchmarkWithPoolParallel(b *testing.B) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		b.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	server := grpc.NewServer()
	pb.RegisterGreeterServer(server, &greeterServer{})

	go server.Serve(l)
	defer server.Stop()

	addr := l.Addr().String()
	dialFunc := func() (*grpc.ClientConn, error) {
		return grpc.Dial(addr, grpc.WithInsecure())
	}

	pool, err := NewPool(10, dialFunc)
	if err != nil {
		return
	}
	defer pool.Close()

	b.ResetTimer()
	b.RunParallel(func(ppb *testing.PB) {
		for ppb.Next() {
			c, err := pool.GetConn()
			if err != nil {
				b.Fatal(err)
			}

			resp := pb.HelloReply{}
			err = c.Invoke(context.TODO(), "/helloworld.Greeter/SayHello", &pb.HelloRequest{Name: "Xin"}, &resp)
			if err != nil {
				b.Fatal(err)
			}
			if resp.Message != "Hello Xin" {
				b.Fatalf("Expect Hello Xin, but got %v", resp.Message)
			}
		}
	})
}
