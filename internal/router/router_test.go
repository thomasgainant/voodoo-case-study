package router_test

import (
	"context"
	"testing"

	pb "voodoo-case-study/gen/voodoo/v1"
	"voodoo-case-study/internal/router"
)

func TestHealthReturnsOK(t *testing.T) {
	r := router.New()
	resp, err := r.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status=ok, got %q", resp.Status)
	}
}

func TestCreateGameReturnsNonEmptyGameID(t *testing.T) {
	r := router.New()
	resp, err := r.CreateGame(context.Background(), &pb.CreateGameRequest{PlayerId: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GameId == "" {
		t.Error("expected a non-empty game_id")
	}
}

func TestCreateGameReturnsDistinctIDs(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	r1, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r2, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p2"})
	if r1.GameId == r2.GameId {
		t.Errorf("expected distinct game IDs, got %q twice", r1.GameId)
	}
}

func TestJoinGameReturnsGameID(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})

	joined, err := r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"})
	if err != nil {
		t.Fatalf("JoinGame failed: %v", err)
	}
	if joined.GameId != created.GameId {
		t.Errorf("expected game_id=%q, got %q", created.GameId, joined.GameId)
	}
}

func TestJoinGameFailsWhenFull(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	_, err := r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p3"})
	if err == nil {
		t.Fatal("expected error when joining a full game")
	}
}

func TestUpdateGameReturnsGameID(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})

	updated, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId})
	if err != nil {
		t.Fatalf("UpdateGame failed: %v", err)
	}
	if updated.GameId != created.GameId {
		t.Errorf("expected game_id=%q, got %q", created.GameId, updated.GameId)
	}
}

func TestUpdateGameFailsForUnknownGame(t *testing.T) {
	r := router.New()
	_, err := r.UpdateGame(context.Background(), &pb.UpdateGameRequest{GameId: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown game")
	}
}
