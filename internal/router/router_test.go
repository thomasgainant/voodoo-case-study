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
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	updated, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0})
	if err != nil {
		t.Fatalf("UpdateGame failed: %v", err)
	}
	if updated.GameId != created.GameId {
		t.Errorf("expected game_id=%q, got %q", created.GameId, updated.GameId)
	}
	if len(updated.Board) != 9 {
		t.Errorf("expected 9 board cells, got %d", len(updated.Board))
	}
}

func TestUpdateGameFailsForUnknownGame(t *testing.T) {
	r := router.New()
	_, err := r.UpdateGame(context.Background(), &pb.UpdateGameRequest{GameId: "unknown", PlayerId: "p1", Cell: 0})
	if err == nil {
		t.Fatal("expected error for unknown game")
	}
}

func TestUpdateGameFailsWhenGameNotReady(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})

	_, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0})
	if err == nil {
		t.Fatal("expected error when game has only one player")
	}
}

func TestUpdateGameFailsWhenNotPlayersTurn(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	_, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 0})
	if err == nil {
		t.Fatal("expected error when p2 plays out of turn")
	}
}

func TestUpdateGameAlternatesTurns(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	type move struct {
		player string
		cell   int32
	}
	// p1: 0, p2: 1, p1: 3, p2: 4 — no winning line
	moves := []move{{"p1", 0}, {"p2", 1}, {"p1", 3}, {"p2", 4}}
	for i, m := range moves {
		if _, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: m.player, Cell: m.cell}); err != nil {
			t.Fatalf("move %d by %q failed: %v", i+1, m.player, err)
		}
	}
}

func TestUpdateGameResponseIncludesWinner(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	// p1 wins top row: cells 0, 1, 2
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 3}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 1}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 4}) //nolint:errcheck
	resp, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 2})
	if err != nil {
		t.Fatalf("final move failed: %v", err)
	}
	if resp.Winner != "p1" {
		t.Errorf("expected winner=p1, got %q", resp.Winner)
	}
}

func TestUpdateGameRejectsOccupiedCell(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 4}) //nolint:errcheck
	_, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 4})
	if err == nil {
		t.Fatal("expected error for occupied cell")
	}
}

func TestListPendingGamesShowsNewGame(t *testing.T) {
	r := router.New()
	ctx := context.Background()

	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})

	resp, err := r.ListPendingGames(ctx, &pb.ListPendingGamesRequest{})
	if err != nil {
		t.Fatalf("ListPendingGames failed: %v", err)
	}
	for _, id := range resp.GameIds {
		if id == created.GameId {
			return
		}
	}
	t.Errorf("expected game %q in pending list, got %v", created.GameId, resp.GameIds)
}

func TestListPendingGamesRemovesGameAfterJoin(t *testing.T) {
	r := router.New()
	ctx := context.Background()

	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	resp, err := r.ListPendingGames(ctx, &pb.ListPendingGamesRequest{})
	if err != nil {
		t.Fatalf("ListPendingGames failed: %v", err)
	}
	for _, id := range resp.GameIds {
		if id == created.GameId {
			t.Errorf("game %q should not be in pending list after join", created.GameId)
		}
	}
}
