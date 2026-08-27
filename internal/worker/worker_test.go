package worker_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"voodoo-case-study/internal/worker"
)

func TestWorkerID(t *testing.T) {
	w := worker.New(3)
	if w.ID() != 3 {
		t.Errorf("expected id=3, got %d", w.ID())
	}
}

func TestCreateGameReturnsStateWithNonEmptyID(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1")
	if state == nil {
		t.Fatal("expected non-nil GameState")
	}
	if !strings.HasPrefix(state.ID, "game-") {
		t.Errorf("unexpected ID format: %q", state.ID)
	}
}

func TestCreateGameGeneratesUniqueIDs(t *testing.T) {
	w := worker.New(0)
	s1 := w.CreateGame("p1")
	s2 := w.CreateGame("p2")
	if s1.ID == s2.ID {
		t.Errorf("expected distinct game IDs, got %q twice", s1.ID)
	}
}

func TestJoinGameAddsSecondPlayer(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1")
	_, err := w.JoinGame(state.ID, "p2")
	if err != nil {
		t.Fatalf("JoinGame failed: %v", err)
	}
}

func TestJoinGameFailsWhenFull(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1")
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	_, err := w.JoinGame(state.ID, "p3")
	if !errors.Is(err, worker.ErrGameFull) {
		t.Errorf("expected ErrGameFull, got %v", err)
	}
}

func TestJoinGameFailsForUnknownGame(t *testing.T) {
	w := worker.New(0)
	_, err := w.JoinGame("no-such-game", "p1")
	if err == nil {
		t.Fatal("expected error for unknown game")
	}
}

func TestWaitReadyUnblocksAfterJoin(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1")

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		done <- state.WaitReady(ctx)
	}()

	w.JoinGame(state.ID, "p2") //nolint:errcheck

	if err := <-done; err != nil {
		t.Errorf("WaitReady returned unexpected error: %v", err)
	}
}

func TestWaitReadyReturnsContextError(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := state.WaitReady(ctx); err == nil {
		t.Error("expected context error, got nil")
	}
}

func TestGetReturnsNilForMissingGameID(t *testing.T) {
	w := worker.New(0)
	if w.Get("missing") != nil {
		t.Error("expected nil for unknown gameID")
	}
}

func TestGetReturnsStateAfterCreateGame(t *testing.T) {
	w := worker.New(0)
	created := w.CreateGame("p1")
	if w.Get(created.ID) != created {
		t.Error("Get should return the same pointer as CreateGame")
	}
}

func TestStateCountReflectsCreatedGames(t *testing.T) {
	w := worker.New(0)
	if w.StateCount() != 0 {
		t.Fatalf("expected 0 states, got %d", w.StateCount())
	}
	w.CreateGame("p1")
	w.CreateGame("p2")
	if w.StateCount() != 2 {
		t.Errorf("expected 2 states, got %d", w.StateCount())
	}
}
