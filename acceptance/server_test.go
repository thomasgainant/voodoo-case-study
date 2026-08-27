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
