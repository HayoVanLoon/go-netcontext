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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/HayoVanLoon/go-netcontext"
	pb "github.com/HayoVanLoon/go-netcontext/examples/go-genproto/netcontext"
	"github.com/HayoVanLoon/go-netcontext/examples/shared"
	nchttp "github.com/HayoVanLoon/go-netcontext/http"
)

type ctxKeyHop struct{}

var CtxKeyHop ctxKeyHop

func init() {
	netcontext.Int32(CtxKeyHop, "hop")
}

type Handler struct {
	grpcClient pb.FooServiceClient
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
		shared.PrintStart(todo, timeout)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}
	if todo <= 0 {
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
	todo = min(todo, math.MaxInt32)
	resp, err = h.grpcClient.Deadline(ctx, &pb.DeadlineRequest{Todo: int32(todo) - 1}) //nolint:gosec
	if err != nil {
		return nil, shared.HandleError(err, hops)
	}
	shared.PrintResponse(hops, resp)
	return resp, err
}

func main() {
	gc, err := shared.DefaultGRPCClient()
	if err != nil {
		log.Fatal(err)
	}
	h := Handler{
		grpcClient: gc,
	}
	http.Handle("/", nchttp.WrapHandler(h))

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
