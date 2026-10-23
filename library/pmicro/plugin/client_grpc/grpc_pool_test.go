package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/status"
)

func testPool(t *testing.T, size int, ttl time.Duration, idle int, ms int) {
	// setup server
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Errorf("failed to listen: %v", err)
	}
	defer l.Close()

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &greeterServer{})

	go s.Serve(l)
	defer s.Stop()

	// zero pool
	p := newPool(size, ttl, idle, ms)

	for i := 0; i < 10; i++ {
		// get a conn
		cc, err := p.getConn(context.TODO(), l.Addr().String(), grpc.WithInsecure())
		if err != nil {
			t.Fatal(err)
		}

		rsp := pb.HelloReply{}

		err = cc.Invoke(context.TODO(), "/helloworld.Greeter/SayHello", &pb.HelloRequest{Name: "John"}, &rsp)
		if err != nil {
			t.Fatal(err)
		}

		if rsp.Message != "Hello John" {
			t.Errorf("Got unexpected response %v", rsp.Message)
		}

		// release the conn
		p.release(l.Addr().String(), cc, nil)

		p.Lock()
		if i := p.conns[l.Addr().String()].count; i > size {
			p.Unlock()
			t.Errorf("pool size %d is greater than expected %d", i, size)
		}
		p.Unlock()
	}
}

func TestGRPCPool(t *testing.T) {
	testPool(t, 0, time.Minute, 10, 2)
	testPool(t, 2, time.Minute, 10, 1)
}

func TestIsConnErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil 错误不关连接", nil, false},
		{"业务错误(Internal)不关连接", status.Error(codes.Internal, "biz error"), false},
		{"NotFound不关连接", status.Error(codes.NotFound, "not found"), false},
		{"服务端超时(DeadlineExceeded)不关连接", status.Error(codes.DeadlineExceeded, "timeout"), false},
		{"非status错误(流级取消)不关连接", context.Canceled, false},
		{"传输不可用(Unavailable)关连接", status.Error(codes.Unavailable, "connection refused"), true},
		{"数据损坏(DataLoss)关连接", status.Error(codes.DataLoss, "data loss"), true},
	}
	for _, c := range cases {
		if got := isConnErr(c.err); got != c.want {
			t.Errorf("%s: isConnErr() = %v, want %v", c.name, got, c.want)
		}
	}
}
