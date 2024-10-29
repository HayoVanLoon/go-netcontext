package shared

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RandomTimer() *time.Timer {
	ms := 250 + rand.Intn(2)*250
	return time.NewTimer(time.Duration(ms) * time.Millisecond)
}

func HandleError(err error, hops int32) error {
	if isContextError(err) {
		PrintDeadlineReached(hops)
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}
	log.Printf("%d: unexpected error: %s", hops, err.Error())
	return status.Error(codes.Internal, err.Error())
}

func isContextError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	switch status.Code(err) {
	case codes.Canceled, codes.DeadlineExceeded:
		return true
	}
	return false
}
