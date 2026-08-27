//go:build acceptance

package acceptance_test

import (
	"context"
	"net"
	"testing"

	pb "voodoo-case-study/gen/voodoo/v1"
	"voodoo-case-study/internal/server"
	"voodoo-case-study/testclient"
)

func newTestServer(t *testing.T) (*testclient.Client, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := server.New()
	go srv.Serve(lis) //nolint:errcheck

	client, err := testclient.New(lis.Addr().String())
	if err != nil {
		srv.Stop()
		t.Fatal(err)
	}
	return client, func() {
		client.Close()  //nolint:errcheck
		srv.Stop()
	}
}

func TestHealthEndpoint(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()

	resp, err := client.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status=ok, got %q", resp.Status)
	}
}

func TestCreateGameEndpoint(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()

	resp, err := client.CreateGame(context.Background(), &pb.CreateGameRequest{PlayerId: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GameId == "" {
		t.Error("expected a non-empty game_id")
	}
}

func TestJoinGameEndpoint(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, err := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	if err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}

	joined, err := client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"})
	if err != nil {
		t.Fatalf("JoinGame failed: %v", err)
	}
	if joined.GameId != created.GameId {
		t.Errorf("expected game_id=%q, got %q", created.GameId, joined.GameId)
	}
}

func TestJoinFullGameReturnsError(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, _ := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	_, err := client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p3"})
	if err == nil {
		t.Fatal("expected error when joining a full game")
	}
}

func TestUpdateGameEndpoint(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, _ := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	updated, err := client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0})
	if err != nil {
		t.Fatalf("UpdateGame failed: %v", err)
	}
	if updated.GameId != created.GameId {
		t.Errorf("expected game_id=%q, got %q", created.GameId, updated.GameId)
	}
}

func TestUpdateGameRejectsWrongTurn(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, _ := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	_, err := client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 0})
	if err == nil {
		t.Fatal("expected error when p2 plays out of turn")
	}
}

func TestUpdateGameRejectsUnreadyGame(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, _ := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})

	_, err := client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0})
	if err == nil {
		t.Fatal("expected error when game has only one player")
	}
}

func TestUpdateGameWinnerIsReturned(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, _ := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	// p1 wins top row: 0, 1, 2
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 3}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 1}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 4}) //nolint:errcheck
	resp, err := client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 2})
	if err != nil {
		t.Fatalf("winning move failed: %v", err)
	}
	if resp.Winner != "p1" {
		t.Errorf("expected winner=p1, got %q", resp.Winner)
	}
	if len(resp.Board) != 9 {
		t.Errorf("expected 9 board cells, got %d", len(resp.Board))
	}
}

func TestListPendingGamesEndpoint(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, err := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	if err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}

	resp, err := client.ListPendingGames(ctx, &pb.ListPendingGamesRequest{})
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

func TestListPendingGamesRemovedAfterJoin(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, _ := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	resp, err := client.ListPendingGames(ctx, &pb.ListPendingGamesRequest{})
	if err != nil {
		t.Fatalf("ListPendingGames failed: %v", err)
	}
	for _, id := range resp.GameIds {
		if id == created.GameId {
			t.Errorf("game %q should be removed from pending list after join", created.GameId)
		}
	}
}

func TestGetPlayerStatsEndpoint(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, _ := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "p1"})
	client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	// p1 wins top row: 0, 1, 2
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 3}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 1}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 4}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 2}) //nolint:errcheck

	p1, err := client.GetPlayerStats(ctx, &pb.GetPlayerStatsRequest{PlayerId: "p1"})
	if err != nil {
		t.Fatalf("GetPlayerStats p1 failed: %v", err)
	}
	if p1.Wins != 1 || p1.Losses != 0 || p1.Draws != 0 {
		t.Errorf("p1: expected wins=1, got wins=%d losses=%d draws=%d", p1.Wins, p1.Losses, p1.Draws)
	}

	p2, err := client.GetPlayerStats(ctx, &pb.GetPlayerStatsRequest{PlayerId: "p2"})
	if err != nil {
		t.Fatalf("GetPlayerStats p2 failed: %v", err)
	}
	if p2.Losses != 1 || p2.Wins != 0 || p2.Draws != 0 {
		t.Errorf("p2: expected losses=1, got wins=%d losses=%d draws=%d", p2.Wins, p2.Losses, p2.Draws)
	}
}

func TestCustomGridGameEndpoint(t *testing.T) {
	client, teardown := newTestServer(t)
	defer teardown()
	ctx := context.Background()

	created, err := client.CreateGame(ctx, &pb.CreateGameRequest{
		PlayerId: "p1",
		Grid:     &pb.GridConfig{Width: 4, Height: 4, WinningLength: 4},
	})
	if err != nil {
		t.Fatalf("CreateGame (4×4) failed: %v", err)
	}

	client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created.GameId, PlayerId: "p2"}) //nolint:errcheck

	// p1 wins top row (cells 0–3) on a 4×4 board with win length 4
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 0}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 4}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 1}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 5}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 2}) //nolint:errcheck
	client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p2", Cell: 6}) //nolint:errcheck
	resp, err := client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: created.GameId, PlayerId: "p1", Cell: 3})
	if err != nil {
		t.Fatalf("winning move failed: %v", err)
	}
	if resp.Winner != "p1" {
		t.Errorf("expected winner=p1 on 4×4 grid, got %q", resp.Winner)
	}
	if len(resp.Board) != 16 {
		t.Errorf("expected 16 board cells for 4×4 grid, got %d", len(resp.Board))
	}
}

