package router

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// CreateGame picks the least-loaded worker, creates a game state with the first player, and
// registers the game ID in the sharder so subsequent calls route to the same worker.
func (r *Router) CreateGame(_ context.Context, req *pb.CreateGameRequest) (*pb.CreateGameResponse, error) {
	w := r.sharder.PickLeastLoaded()
	state := w.CreateGame(req.PlayerId)
	r.sharder.Register(state.ID, w)
	return &pb.CreateGameResponse{GameId: state.ID}, nil
}

func (r *Router) JoinGame(_ context.Context, req *pb.JoinGameRequest) (*pb.JoinGameResponse, error) {
	w := r.sharder.Resolve(req.GameId)
	if _, err := w.JoinGame(req.GameId, req.PlayerId); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
	}
	return &pb.JoinGameResponse{GameId: req.GameId}, nil
}

func (r *Router) UpdateGame(_ context.Context, req *pb.UpdateGameRequest) (*pb.UpdateGameResponse, error) {
	w := r.sharder.Resolve(req.GameId)
	if w.Get(req.GameId) == nil {
		return nil, status.Errorf(codes.NotFound, "game %q not found", req.GameId)
	}
	return &pb.UpdateGameResponse{GameId: req.GameId}, nil
}
