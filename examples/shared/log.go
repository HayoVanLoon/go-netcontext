package shared

import (
	"context"
	"fmt"
	pb "github.com/HayoVanLoon/go-netcontext/examples/go-genproto/netcontext"
	"log"
	"strings"
	"time"
)

func PrintStart[Int int | int32](todo, timeout Int) {
	t := time.Now().Add(time.Duration(timeout) * time.Second)
	log.Printf("starting %d hops with deadline set to %s", todo, t.Format(time.TimeOnly))
}

func PrintHop[Int int | int32](hops Int) {
	log.Printf(prefix(hops) + "calling other service")
}

func PrintResponse[Int int | int32](hops Int, resp *pb.DeadlineResponse) {
	log.Printf(prefix(hops)+"response with %d hops", resp.Hops)
}

func PrintDone[Int int | int32](ctx context.Context, hops Int) {
	t, _ := ctx.Deadline()
	log.Printf(prefix(hops)+"done with %v seconds left", t.Sub(time.Now()).Seconds())
}

func PrintDeadlineReached[Int int | int32](hops Int) {
	log.Printf(prefix(hops) + "deadline reached")
}

func prefix[Int int | int32](hops Int) string {
	indent := strings.Repeat(" ", int(hops))
	return fmt.Sprintf("%s%d: ", indent, hops)
}
