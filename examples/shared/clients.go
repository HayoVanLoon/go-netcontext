package shared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/HayoVanLoon/go-netcontext/examples/go-genproto/netcontext"
)

var Host = "localhost"
var HTTPPort = "8080"
var GRPCPort = "8081"
var Target = Host + ":" + GRPCPort

type ExampleHTTPClient struct {
	host string
	*http.Client
}

type ExampleResponse struct{}

func (c *ExampleHTTPClient) Deadline(ctx context.Context, todo, timeout int32) (*pb.DeadlineResponse, error) {
	u := fmt.Sprintf("http://%s/deadline", c.host)
	q := url.Values{}
	if todo > 0 {
		q.Set("todo", strconv.FormatInt(int64(todo), 10))
	}
	if timeout > 0 {
		q.Set("timeout", strconv.FormatInt(int64(timeout), 10))
	}
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	resp := &pb.DeadlineResponse{}
	if err := c.call(ctx, u, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExampleHTTPClient) call(ctx context.Context, u string, a any) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	resp, err := c.Do(r)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return status.Errorf(codes.DeadlineExceeded, err.Error())
		}
		return status.Errorf(codes.Internal, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		bs, _ := io.ReadAll(resp.Body)
		st := new(spb.Status)
		if err = protojson.Unmarshal(bs, st); err == nil {
			return status.ErrorProto(st)
		}
		return status.Errorf(codes.Internal, string(bs))
	}
	if a != nil {
		err = json.NewDecoder(resp.Body).Decode(&a)
	}
	return err
}

func NewExampleHTTPClient(client *http.Client) *ExampleHTTPClient {
	return &ExampleHTTPClient{
		host:   Host + ":" + HTTPPort,
		Client: client,
	}
}
