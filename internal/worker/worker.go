package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var ErrGameFull     = errors.New("game is full")
var ErrGameNotReady = errors.New("game is not ready")
var ErrNotYourTurn  = errors.New("not your turn")
var ErrCellOccupied = errors.New("cell is already occupied")
var ErrInvalidCell  = errors.New("cell index out of range")
var ErrGameOver     = errors.New("game is already over")

// winLines lists the eight winning lines on a 3×3 board (row-major, 0-indexed).
var winLines = [8][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // cols
	{0, 4, 8}, {2, 4, 6},             // diagonals
}

// GameState holds the in-memory state for a single game.
type GameState struct {
	ID          string
	mu          sync.Mutex
	players     [2]string
	playerCount int
	currentTurn int          // index (0 or 1) of the player whose turn it is
	board       [9]int8      // 0=empty, 1=players[0] mark, 2=players[1] mark
	winner      int8         // 0=ongoing, 1=players[0] won, 2=players[1] won, 3=draw
	ready       chan struct{} // closed when both player slots are filled
}

func newGameState(id, firstPlayerID string) *GameState {
	g := &GameState{
		ID:          id,
		playerCount: 1,
		ready:       make(chan struct{}),
	}
	g.players[0] = firstPlayerID
	return g
}

// AddPlayer fills the second player slot. Returns ErrGameFull if both slots are taken.
func (g *GameState) AddPlayer(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.playerCount >= 2 {
		return ErrGameFull
	}
	g.players[1] = playerID
	g.playerCount++
	close(g.ready)
	return nil
}

// TakeTurn places the current player's mark on cell (0-8) and detects win/draw.
func (g *GameState) TakeTurn(playerID string, cell int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.playerCount < 2 {
		return ErrGameNotReady
	}
	if g.winner != 0 {
		return ErrGameOver
	}
	if g.players[g.currentTurn] != playerID {
		return ErrNotYourTurn
	}
	if cell < 0 || cell > 8 {
		return ErrInvalidCell
	}
	if g.board[cell] != 0 {
		return ErrCellOccupied
	}

	mark := int8(g.currentTurn + 1)
	g.board[cell] = mark

	if g.hasWon(mark) {
		g.winner = mark
	} else if g.isBoardFull() {
		g.winner = 3 // draw
	} else {
		g.currentTurn = 1 - g.currentTurn
	}
	return nil
}

func (g *GameState) hasWon(mark int8) bool {
	for _, line := range winLines {
		if g.board[line[0]] == mark && g.board[line[1]] == mark && g.board[line[2]] == mark {
			return true
		}
	}
	return false
}

func (g *GameState) isBoardFull() bool {
	for _, v := range g.board {
		if v == 0 {
			return false
		}
	}
	return true
}

// Board returns the 9-cell board with player IDs (empty string for empty cells).
func (g *GameState) Board() [9]string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out [9]string
	for i, v := range g.board {
		if v == 1 {
			out[i] = g.players[0]
		} else if v == 2 {
			out[i] = g.players[1]
		}
	}
	return out
}

// Winner returns "" if ongoing, the winning player_id if won, or "draw".
func (g *GameState) Winner() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch g.winner {
	case 1:
		return g.players[0]
	case 2:
		return g.players[1]
	case 3:
		return "draw"
	default:
		return ""
	}
}

// WaitReady blocks until both players have joined or ctx is done.
func (g *GameState) WaitReady(ctx context.Context) error {
	select {
	case <-g.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Worker processes all requests for its assigned game IDs.
type Worker struct {
	id     int
	mu     sync.RWMutex
	states map[string]*GameState
	nextID atomic.Uint64
}

func New(id int) *Worker {
	return &Worker{
		id:     id,
		states: make(map[string]*GameState),
	}
}

func (w *Worker) ID() int { return w.id }

// CreateGame creates a new game state with playerID as the first player.
func (w *Worker) CreateGame(playerID string) *GameState {
	id := fmt.Sprintf("game-%d-%d", w.id, w.nextID.Add(1))
	state := newGameState(id, playerID)
	w.mu.Lock()
	w.states[id] = state
	w.mu.Unlock()
	return state
}

// JoinGame adds playerID as the second player of an existing game.
// Returns ErrGameFull if the game already has two players.
func (w *Worker) JoinGame(gameID, playerID string) (*GameState, error) {
	w.mu.RLock()
	state, ok := w.states[gameID]
	w.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("game %q not found", gameID)
	}
	if err := state.AddPlayer(playerID); err != nil {
		return nil, err
	}
	return state, nil
}

// UpdateGame places playerID's mark on cell, enforcing turn order and game rules.
// Returns the updated GameState so callers can read board and winner.
func (w *Worker) UpdateGame(gameID, playerID string, cell int) (*GameState, error) {
	w.mu.RLock()
	state, ok := w.states[gameID]
	w.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("game %q not found", gameID)
	}
	if err := state.TakeTurn(playerID, cell); err != nil {
		return nil, err
	}
	return state, nil
}

// Get returns the GameState for the given gameID, or nil if not found.
func (w *Worker) Get(gameID string) *GameState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.states[gameID]
}

// StateCount returns the number of game states currently held by the worker.
func (w *Worker) StateCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.states)
}

