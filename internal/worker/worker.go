package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// ErrGameFull is returned when a player attempts to join an already full game.
var ErrGameFull = errors.New("game is full")

// GameState holds the in-memory state for a single game.
type GameState struct {
	ID          string
	mu          sync.Mutex
	players     [2]string
	playerCount int
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

