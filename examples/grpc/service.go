package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/HayoVanLoon/go-netcontext"
	pb "github.com/HayoVanLoon/go-netcontext/examples/go-genproto/netcontext"
	"github.com/HayoVanLoon/go-netcontext/examples/shared"
	ncgrpc "github.com/HayoVanLoon/go-netcontext/grpc"
	nchttp "github.com/HayoVanLoon/go-netcontext/http"
	"google.golang.org/grpc"
)

type ctxKeyHop struct{}

var CtxKeyHop ctxKeyHop

func init() {
	// Configure context value (on load). See the HTTP example service for
	// configuring values on start-up.
	netcontext.Int32(CtxKeyHop, "hop")
}

type ExampleService struct {
	pb.UnimplementedExampleServiceServer
	client *shared.ExampleHTTPClient
}

func (ex *ExampleService) Deadline(ctx context.Context, req *pb.DeadlineRequest) (resp *pb.DeadlineResponse, err error) {
	hops, _ := ctx.Value(CtxKeyHop).(int32)
	ctx = context.WithValue(ctx, CtxKeyHop, hops+1)

	if _, ok := ctx.Deadline(); !ok {
		// Start the process.
		shared.PrintStart(req.Todo, req.Timeout)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}
	if req.Todo <= 0 {
		// Success.
		shared.PrintDone(ctx, hops)
		return &pb.DeadlineResponse{Hops: hops}, nil
	}

	sleep := shared.RandomTimer()
	defer sleep.Stop()
	select {
	case <-ctx.Done():
		// Time's up.
		return nil, status.Error(codes.DeadlineExceeded, ctx.Err().Error())
	case <-sleep.C:
	}

	// Call the other service.
	shared.PrintHop(hops)
	resp, err = ex.client.Deadline(ctx, req.Todo-1, 0)
	if err != nil {
		return nil, shared.HandleError(err, hops)
	}
	shared.PrintResponse(hops, resp)
	return resp, err
}

func main() {
	// Use the interceptor for incoming requests.
	srv := grpc.NewServer(grpc.UnaryInterceptor(ncgrpc.UnaryServerIntercept))
	// Wrap the default http.Client for outgoing requests.
	httpClient := nchttp.WrapClient(http.DefaultClient)
	client := shared.NewExampleHTTPClient(httpClient)
	svc := &ExampleService{
		client: client,
	}

	// The rest of the server initialisation boilerplate...
	pb.RegisterExampleServiceServer(srv, svc)

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
