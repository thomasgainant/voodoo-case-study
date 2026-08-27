package router

import (
	"context"

	pb "voodoo-case-study/gen/voodoo/v1"
	"voodoo-case-study/internal/sharder"
)

// Router dispatches incoming gRPC requests to the appropriate worker.
// Each RPC method delegates to the Worker resolved by the Sharder.
type Router struct {
	pb.UnimplementedVoodooServiceServer
	sharder *sharder.Sharder
}

func New() *Router {
	return &Router{
		sharder: sharder.New(8),
	}
}

func (r *Router) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "ok"}, nil
}
