package main

import (
	"context"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/HayoVanLoon/go-netcontext"
	pb "github.com/HayoVanLoon/go-netcontext/examples/go-genproto/netcontext"
	"github.com/HayoVanLoon/go-netcontext/examples/shared"
	ncgrpc "github.com/HayoVanLoon/go-netcontext/grpc"
	nchttp "github.com/HayoVanLoon/go-netcontext/http"
)

type ctxKeyHop struct{}

var CtxKeyHop ctxKeyHop

func init() {
}

type Handler struct {
	client pb.ExampleServiceClient
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// skip validation
	todo, _ := strconv.Atoi(r.URL.Query().Get("todo"))
	timeout, _ := strconv.Atoi(r.URL.Query().Get("timeout"))

	var out []byte
	resp, err := h.Deadline(r.Context(), todo, timeout)
	e, _ := status.FromError(err)
	switch e.Code() {
	case codes.OK:
		w.WriteHeader(http.StatusOK)
		out, _ = protojson.Marshal(resp)
	case codes.DeadlineExceeded:
		w.WriteHeader(http.StatusRequestTimeout)
		out, _ = protojson.Marshal(e.Proto())
	default:
		w.WriteHeader(http.StatusInternalServerError)
		out, _ = protojson.Marshal(e.Proto())
	}
	_, _ = w.Write(out)
}

func (h Handler) Deadline(ctx context.Context, todo, timeout int) (resp *pb.DeadlineResponse, err error) {
	hops, _ := ctx.Value(CtxKeyHop).(int32)
	ctx = context.WithValue(ctx, CtxKeyHop, hops+1)

	if _, ok := ctx.Deadline(); !ok {
		// Start the process.
		shared.PrintStart(todo, timeout)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}
	if todo <= 0 {
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
	todo = min(todo, math.MaxInt32)
	resp, err = h.client.Deadline(ctx, &pb.DeadlineRequest{Todo: int32(todo) - 1}) //nolint:gosec
	if err != nil {
		return nil, shared.HandleError(err, hops)
	}
	shared.PrintResponse(hops, resp)
	return resp, err
}

func main() {
	// Configure context value (on start-up). See the gRPC example service for
	// configuring values on package load.
	netcontext.Int32(CtxKeyHop, "hop")

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Use the client interceptor for outgoing requests.
		grpc.WithUnaryInterceptor(ncgrpc.UnaryClientIntercept),
	}
	conn, err := grpc.NewClient(shared.Target, opts...)
	if err != nil {
		log.Fatal(err)
	}
	client := pb.NewExampleServiceClient(conn)

	// Wrap the Handler for incoming requests.
	h := nchttp.WrapHandler(Handler{
		client: client,
	})

	// The rest of the server initialisation boilerplate...
	http.Handle("/", h)

	go func() {
		log.Println("server running")
		if err = http.ListenAndServe(":8080", http.DefaultServeMux); err != nil {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down")
}
