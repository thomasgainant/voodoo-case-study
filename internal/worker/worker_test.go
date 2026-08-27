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
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	if state == nil {
		t.Fatal("expected non-nil GameState")
	}
	if !strings.HasPrefix(state.ID, "game-") {
		t.Errorf("unexpected ID format: %q", state.ID)
	}
}

func TestCreateGameGeneratesUniqueIDs(t *testing.T) {
	w := worker.New(0)
	s1 := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	s2 := w.CreateGame("p2", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	if s1.ID == s2.ID {
		t.Errorf("expected distinct game IDs, got %q twice", s1.ID)
	}
}

func TestJoinGameAddsSecondPlayer(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	_, err := w.JoinGame(state.ID, "p2")
	if err != nil {
		t.Fatalf("JoinGame failed: %v", err)
	}
}

func TestJoinGameFailsWhenFull(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
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
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})

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
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})

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
	created := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	if w.Get(created.ID) != created {
		t.Error("Get should return the same pointer as CreateGame")
	}
}

func TestStateCountReflectsCreatedGames(t *testing.T) {
	w := worker.New(0)
	if w.StateCount() != 0 {
		t.Fatalf("expected 0 states, got %d", w.StateCount())
	}
	w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.CreateGame("p2", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	if w.StateCount() != 2 {
		t.Errorf("expected 2 states, got %d", w.StateCount())
	}
}

func TestUpdateGameFailsWhenGameNotReady(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	if _, err := w.UpdateGame(state.ID, "p1", 0); !errors.Is(err, worker.ErrGameNotReady) {
		t.Errorf("expected ErrGameNotReady, got %v", err)
	}
}

func TestUpdateGameFailsForUnknownGame(t *testing.T) {
	w := worker.New(0)
	if _, err := w.UpdateGame("no-such-game", "p1", 0); err == nil {
		t.Fatal("expected error for unknown game")
	}
}

func TestUpdateGameFailsWhenNotPlayersTurn(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	if _, err := w.UpdateGame(state.ID, "p2", 0); !errors.Is(err, worker.ErrNotYourTurn) {
		t.Errorf("expected ErrNotYourTurn, got %v", err)
	}
}

func TestUpdateGameSucceedsForCurrentPlayer(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	if _, err := w.UpdateGame(state.ID, "p1", 0); err != nil {
		t.Errorf("expected p1 to succeed on first turn, got %v", err)
	}
}

func TestUpdateGameAlternatesTurns(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	// p1: 0, p2: 1, p1: 3, p2: 4 — no winning line
	type move struct {
		player string
		cell   int
	}
	moves := []move{{"p1", 0}, {"p2", 1}, {"p1", 3}, {"p2", 4}}
	for i, m := range moves {
		if _, err := w.UpdateGame(state.ID, m.player, m.cell); err != nil {
			t.Fatalf("move %d by %q failed: %v", i+1, m.player, err)
		}
	}
}

func TestUpdateGameFailsForOccupiedCell(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	w.UpdateGame(state.ID, "p1", 4) //nolint:errcheck
	if _, err := w.UpdateGame(state.ID, "p2", 4); !errors.Is(err, worker.ErrCellOccupied) {
		t.Errorf("expected ErrCellOccupied, got %v", err)
	}
}

func TestUpdateGameFailsForInvalidCell(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	if _, err := w.UpdateGame(state.ID, "p1", 9); !errors.Is(err, worker.ErrInvalidCell) {
		t.Errorf("expected ErrInvalidCell, got %v", err)
	}
}

func TestWinDetectionRow(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	// p1 wins top row: 0,1,2
	w.UpdateGame(state.ID, "p1", 0) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 3) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 1) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 4) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 2) //nolint:errcheck

	if state.Winner() != "p1" {
		t.Errorf("expected winner p1, got %q", state.Winner())
	}
}

func TestWinDetectionCol(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	// p1 wins left col: 0,3,6
	w.UpdateGame(state.ID, "p1", 0) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 1) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 3) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 2) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 6) //nolint:errcheck

	if state.Winner() != "p1" {
		t.Errorf("expected winner p1, got %q", state.Winner())
	}
}

func TestWinDetectionDiagonal(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	// p1 wins main diagonal: 0,4,8
	w.UpdateGame(state.ID, "p1", 0) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 1) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 4) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 2) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 8) //nolint:errcheck

	if state.Winner() != "p1" {
		t.Errorf("expected winner p1, got %q", state.Winner())
	}
}

func TestDrawDetection(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	// Final board (no winning line):
	// p1 O p1
	// p1 p1 p2
	// p2 p1 p2
	moves := [][2]interface{}{
		{"p1", 0}, {"p2", 1}, {"p1", 2},
		{"p2", 5}, {"p1", 3}, {"p2", 6},
		{"p1", 4}, {"p2", 8}, {"p1", 7},
	}
	for i, m := range moves {
		if _, err := w.UpdateGame(state.ID, m[0].(string), m[1].(int)); err != nil {
			t.Fatalf("move %d failed: %v", i+1, err)
		}
	}

	if state.Winner() != "draw" {
		t.Errorf("expected draw, got %q", state.Winner())
	}
}

func TestMovesRejectedAfterGameOver(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	// p1 wins: top row
	w.UpdateGame(state.ID, "p1", 0) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 3) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 1) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 4) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 2) //nolint:errcheck

	if _, err := w.UpdateGame(state.ID, "p2", 5); !errors.Is(err, worker.ErrGameOver) {
		t.Errorf("expected ErrGameOver, got %v", err)
	}
}

func TestBoardReflectsMoves(t *testing.T) {
	w := worker.New(0)
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	w.UpdateGame(state.ID, "p1", 4) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 0) //nolint:errcheck

	board := state.Board()
	if board[4] != "p1" {
		t.Errorf("expected cell 4 = p1, got %q", board[4])
	}
	if board[0] != "p2" {
		t.Errorf("expected cell 0 = p2, got %q", board[0])
	}
	if board[1] != "" {
		t.Errorf("expected cell 1 empty, got %q", board[1])
	}
}

func TestWinDetectionRowOnLargeGrid(t *testing.T) {
	// 4×4 grid, win length 4: p1 wins by filling the top row (cells 0–3)
	w := worker.New(0)
	cfg := worker.GridConfig{Width: 4, Height: 4, WinLen: 4}
	state := w.CreateGame("p1", cfg)
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	w.UpdateGame(state.ID, "p1", 0) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 4) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 1) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 5) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 2) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 6) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 3) //nolint:errcheck

	if state.Winner() != "p1" {
		t.Errorf("expected winner p1, got %q", state.Winner())
	}
}

func TestWinDetectionColOnLargeGrid(t *testing.T) {
	// 4×4 grid, win length 4: p1 wins column 0 (cells 0,4,8,12)
	w := worker.New(0)
	cfg := worker.GridConfig{Width: 4, Height: 4, WinLen: 4}
	state := w.CreateGame("p1", cfg)
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	w.UpdateGame(state.ID, "p1", 0)  //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 1)  //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 4)  //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 5)  //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 8)  //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 9)  //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 12) //nolint:errcheck

	if state.Winner() != "p1" {
		t.Errorf("expected winner p1, got %q", state.Winner())
	}
}

func TestWinDetectionDiagonalOnLargeGrid(t *testing.T) {
	// 4×4 grid, win length 4: p1 wins main diagonal (cells 0,5,10,15)
	w := worker.New(0)
	cfg := worker.GridConfig{Width: 4, Height: 4, WinLen: 4}
	state := w.CreateGame("p1", cfg)
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	w.UpdateGame(state.ID, "p1", 0)  //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 1)  //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 5)  //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 2)  //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 10) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 3)  //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 15) //nolint:errcheck

	if state.Winner() != "p1" {
		t.Errorf("expected winner p1, got %q", state.Winner())
	}
}

func TestThreeInARowNotWinOnWinLength4Grid(t *testing.T) {
	// 4×4 grid, win length 4: three marks in a row must not trigger a win
	w := worker.New(0)
	cfg := worker.GridConfig{Width: 4, Height: 4, WinLen: 4}
	state := w.CreateGame("p1", cfg)
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	w.UpdateGame(state.ID, "p1", 0) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 4) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 1) //nolint:errcheck
	w.UpdateGame(state.ID, "p2", 5) //nolint:errcheck
	w.UpdateGame(state.ID, "p1", 2) //nolint:errcheck

	if state.Winner() != "" {
		t.Errorf("expected no winner after 3 in a row on win-4 grid, got %q", state.Winner())
	}
}

func TestInvalidCellOnNonSquareGrid(t *testing.T) {
	// 5×3 grid has 15 cells (0–14); cell 15 is out of range
	w := worker.New(0)
	cfg := worker.GridConfig{Width: 5, Height: 3, WinLen: 4}
	state := w.CreateGame("p1", cfg)
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	if _, err := w.UpdateGame(state.ID, "p1", 15); !errors.Is(err, worker.ErrInvalidCell) {
		t.Errorf("expected ErrInvalidCell for cell 15 on 5×3 grid, got %v", err)
	}
}

func TestBoardSizeMatchesGridConfig(t *testing.T) {
	// A 4×5 grid should produce a 20-cell board
	w := worker.New(0)
	cfg := worker.GridConfig{Width: 4, Height: 5, WinLen: 3}
	state := w.CreateGame("p1", cfg)
	w.JoinGame(state.ID, "p2") //nolint:errcheck

	w.UpdateGame(state.ID, "p1", 0) //nolint:errcheck

	board := state.Board()
	if len(board) != 20 {
		t.Errorf("expected board size 20 for 4×5 grid, got %d", len(board))
	}
	if board[0] != "p1" {
		t.Errorf("expected board[0]=p1, got %q", board[0])
	}
}

