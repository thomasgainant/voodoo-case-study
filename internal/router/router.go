package router

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "voodoo-case-study/gen/voodoo/v1"
	"voodoo-case-study/internal/sharder"
	"voodoo-case-study/internal/stats"
	"voodoo-case-study/internal/worker"
)

// Router dispatches incoming gRPC requests to the appropriate worker.
// Each RPC method delegates to the Worker resolved by the Sharder.
type Router struct {
	pb.UnimplementedVoodooServiceServer
	sharder *sharder.Sharder
	stats   *stats.Store
}

func New() *Router {
	return &Router{
		sharder: sharder.New(8, 10_000),
		stats:   stats.New(),
	}
}

func (r *Router) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "ok"}, nil
}

// CreateGame picks the least-loaded worker, creates a game state with the first player, and
// registers the game ID in the sharder so subsequent calls route to the same worker.
func (r *Router) CreateGame(_ context.Context, req *pb.CreateGameRequest) (*pb.CreateGameResponse, error) {
	cfg := worker.GridConfig{Width: 3, Height: 3, WinLen: 3}
	if g := req.Grid; g != nil && g.Width > 0 && g.Height > 0 && g.WinningLength > 0 {
		cfg = worker.GridConfig{Width: int(g.Width), Height: int(g.Height), WinLen: int(g.WinningLength)}
	}
	w := r.sharder.PickLeastLoaded()
	state := w.CreateGame(req.PlayerId, cfg)
	r.sharder.Register(state.ID, w)
	r.sharder.AddPending(state.ID)
	return &pb.CreateGameResponse{GameId: state.ID}, nil
}

func (r *Router) JoinGame(_ context.Context, req *pb.JoinGameRequest) (*pb.JoinGameResponse, error) {
	w := r.sharder.Resolve(req.GameId)
	if _, err := w.JoinGame(req.GameId, req.PlayerId); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
	}
	r.sharder.RemovePending(req.GameId)
	return &pb.JoinGameResponse{GameId: req.GameId}, nil
}

func (r *Router) UpdateGame(_ context.Context, req *pb.UpdateGameRequest) (*pb.UpdateGameResponse, error) {
	w := r.sharder.Resolve(req.GameId)
	state, err := w.UpdateGame(req.GameId, req.PlayerId, int(req.Cell))
	if err != nil {
		if errors.Is(err, worker.ErrNotYourTurn) || errors.Is(err, worker.ErrGameNotReady) ||
			errors.Is(err, worker.ErrCellOccupied) || errors.Is(err, worker.ErrInvalidCell) ||
			errors.Is(err, worker.ErrGameOver) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		}
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	board := state.Board()
	winner := state.Winner()
	if winner != "" {
		players := state.Players()
		if winner == "draw" {
			r.stats.RecordDraw(players[0])
			r.stats.RecordDraw(players[1])
		} else {
			loser := players[0]
			if players[0] == winner {
				loser = players[1]
			}
			r.stats.RecordWin(winner)
			r.stats.RecordLoss(loser)
		}
	}
	return &pb.UpdateGameResponse{
		GameId: req.GameId,
		Board:  board,
		Winner: winner,
	}, nil
}

func (r *Router) ListPendingGames(_ context.Context, _ *pb.ListPendingGamesRequest) (*pb.ListPendingGamesResponse, error) {
	return &pb.ListPendingGamesResponse{GameIds: r.sharder.ListPending()}, nil
}

func (r *Router) GetPlayerStats(_ context.Context, req *pb.GetPlayerStatsRequest) (*pb.GetPlayerStatsResponse, error) {
	rec := r.stats.Get(req.PlayerId)
	return &pb.GetPlayerStatsResponse{
		Wins:   rec.Wins,
		Losses: rec.Losses,
		Draws:  rec.Draws,
	}, nil
}
