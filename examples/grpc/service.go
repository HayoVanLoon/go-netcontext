package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/HayoVanLoon/go-netcontext"
	pb "github.com/HayoVanLoon/go-netcontext/examples/go-genproto/netcontext"
	"github.com/HayoVanLoon/go-netcontext/examples/shared"
	ncgrpc "github.com/HayoVanLoon/go-netcontext/grpc"
)

type ctxKeyHop struct{}

var CtxKeyHop ctxKeyHop

func init() {
	netcontext.Int32(CtxKeyHop, "hop")
}

type FooService struct {
	pb.UnimplementedFooServiceServer
	httpClient *shared.FooHTTPClient
}

func (foo *FooService) Deadline(ctx context.Context, req *pb.DeadlineRequest) (resp *pb.DeadlineResponse, err error) {
	hops, _ := ctx.Value(CtxKeyHop).(int32)
	ctx = context.WithValue(ctx, CtxKeyHop, hops+1)

	if _, ok := ctx.Deadline(); !ok {
		shared.PrintStart(req.Todo, req.Timeout)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}
	if req.Todo <= 0 {
		shared.PrintDone(ctx, hops)
		return &pb.DeadlineResponse{Hops: hops}, nil
	}

	sleep := shared.RandomTimer()
	defer sleep.Stop()
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, ctx.Err().Error())
	case <-sleep.C:
	}

	shared.PrintHop(hops)
	resp, err = foo.httpClient.Deadline(ctx, req.Todo-1, 0)
	if err != nil {
		return nil, shared.HandleError(err, hops)
	}
	shared.PrintResponse(hops, resp)
	return resp, err
}

func main() {
	srv := grpc.NewServer(grpc.UnaryInterceptor(ncgrpc.UnaryServerIntercept))
	svc := &FooService{
		httpClient: shared.DefaultHTTPClient(),
	}
	pb.RegisterFooServiceServer(srv, svc)

	lis, err := net.Listen("tcp", ":"+shared.GRPCPort)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Println("server running")
		if err = srv.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down")
}
