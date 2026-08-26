package server_test

import (
	"context"
	"net"
	"testing"

	pb "voodoo-case-study/gen/voodoo/v1"
	"voodoo-case-study/internal/server"
	"voodoo-case-study/testclient"
)

func startServer(t *testing.T) *testclient.Client {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := server.New()
	go srv.Serve(lis) //nolint:errcheck
	t.Cleanup(srv.Stop)
	client, err := testclient.New(lis.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestHealthReturnsOK(t *testing.T) {
	client := startServer(t)

	resp, err := client.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status=ok, got %q", resp.Status)
	}
}
