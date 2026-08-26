//go:build acceptance

package acceptance_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"voodoo-case-study/internal/server"
)

func TestHealthEndpoint(t *testing.T) {
	ts := httptest.NewServer(server.New())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
}
