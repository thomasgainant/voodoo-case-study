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

func TestGetPlayerStatsZeroForNewPlayer(t *testing.T) {
	r := router.New()
	resp, err := r.GetPlayerStats(context.Background(), &pb.GetPlayerStatsRequest{PlayerId: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Wins != 0 || resp.Losses != 0 || resp.Draws != 0 {
		t.Errorf("expected all zeros, got wins=%d losses=%d draws=%d", resp.Wins, resp.Losses, resp.Draws)
	}
}

func TestGetPlayerStatsAfterWin(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	// p1 wins top row: 0, 1, 2
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 3}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 1}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 4}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 2}) //nolint:errcheck

	p1, _ := r.GetPlayerStats(ctx, &pb.GetPlayerStatsRequest{PlayerId: "p1"})
	if p1.Wins != 1 || p1.Losses != 0 {
		t.Errorf("p1: expected wins=1 losses=0, got wins=%d losses=%d", p1.Wins, p1.Losses)
	}
	p2, _ := r.GetPlayerStats(ctx, &pb.GetPlayerStatsRequest{PlayerId: "p2"})
	if p2.Losses != 1 || p2.Wins != 0 {
		t.Errorf("p2: expected losses=1 wins=0, got wins=%d losses=%d", p2.Wins, p2.Losses)
	}
}

func TestGetPlayerStatsAfterDraw(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	// draw board: p1=0,2,3,4,7 p2=1,5,6,8
	type move struct {
		player string
		cell   int32
	}
	for _, m := range []move{
		{"p1", 0}, {"p2", 1}, {"p1", 2},
		{"p2", 5}, {"p1", 3}, {"p2", 6},
		{"p1", 4}, {"p2", 8}, {"p1", 7},
	} {
		r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: m.player, Cell: m.cell}) //nolint:errcheck
	}

	for _, playerID := range []string{"p1", "p2"} {
		resp, _ := r.GetPlayerStats(ctx, &pb.GetPlayerStatsRequest{PlayerId: playerID})
		if resp.Draws != 1 || resp.Wins != 0 || resp.Losses != 0 {
			t.Errorf("%s: expected draws=1, got wins=%d losses=%d draws=%d", playerID, resp.Wins, resp.Losses, resp.Draws)
		}
	}
}

func TestCreateGameWithCustomGrid(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	resp, err := r.CreateGame(ctx, &pb.CreateGameRequest{
		PlayerId: "p1",
		Grid:     &pb.GridConfig{Width: 4, Height: 4, WinningLength: 4},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GameId == "" {
		t.Error("expected a non-empty game_id")
	}
}

func TestUpdateGameBoardSizeMatchesCustomGrid(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{
		PlayerId: "p1",
		Grid:     &pb.GridConfig{Width: 4, Height: 4, WinningLength: 4},
	})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	updated, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0})
	if err != nil {
		t.Fatalf("UpdateGame failed: %v", err)
	}
	if len(updated.Board) != 16 {
		t.Errorf("expected 16 board cells for 4×4 grid, got %d", len(updated.Board))
	}
}

func TestUpdateGameWinnerOnCustomGrid(t *testing.T) {
	r := router.New()
	ctx := context.Background()
	created, _ := r.CreateGame(ctx, &pb.CreateGameRequest{
		PlayerId: "p1",
		Grid:     &pb.GridConfig{Width: 4, Height: 4, WinningLength: 4},
	})
	r.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	// p1 wins top row (cells 0–3) on a 4×4 board with win length 4
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 4}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 1}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 5}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 2}) //nolint:errcheck
	r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 6}) //nolint:errcheck
	resp, err := r.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 3})
	if err != nil {
		t.Fatalf("final move failed: %v", err)
	}
	if resp.Winner != "p1" {
		t.Errorf("expected winner=p1 on 4×4 grid, got %q", resp.Winner)
	}
}
