package router

import (
	"context"

	pb "voodoo-case-study/gen/voodoo/v1"
)

// Router dispatches incoming gRPC requests to the appropriate worker.
// Each RPC method will delegate to a dedicated worker once implemented.
type Router struct {
	pb.UnimplementedVoodooServiceServer
}

func New() *Router {
	return &Router{}
}

func (r *Router) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "ok"}, nil
}
