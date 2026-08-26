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

func TestHealthEndpoint(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := server.New()
	go srv.Serve(lis) //nolint:errcheck
	defer srv.Stop()

	client, err := testclient.New(lis.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	resp, err := client.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status=ok, got %q", resp.Status)
	}
}
