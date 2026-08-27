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

// GridConfig defines the board dimensions and win condition.
type GridConfig struct {
	Width, Height, WinLen int
}

// defaultGrid is the classic 3×3 Tic-Tac-Toe configuration.
var defaultGrid = GridConfig{Width: 3, Height: 3, WinLen: 3}

// GameState holds the in-memory state for a single game.
type GameState struct {
	ID          string
	mu          sync.Mutex
	players     [2]string
	playerCount int
	currentTurn int          // index (0 or 1) of the player whose turn it is
	width       int
	height      int
	winLen      int
	board       []int8       // 0=empty, 1=players[0] mark, 2=players[1] mark
	winner      int8         // 0=ongoing, 1=players[0] won, 2=players[1] won, 3=draw
	ready       chan struct{} // closed when both player slots are filled
}

func newGameState(id, firstPlayerID string, cfg GridConfig) *GameState {
	g := &GameState{
		ID:          id,
		playerCount: 1,
		width:       cfg.Width,
		height:      cfg.Height,
		winLen:      cfg.WinLen,
		board:       make([]int8, cfg.Width*cfg.Height),
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

// TakeTurn places the current player's mark on cell and detects win/draw.
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
	if cell < 0 || cell >= g.width*g.height {
		return ErrInvalidCell
	}
	if g.board[cell] != 0 {
		return ErrCellOccupied
	}

	mark := int8(g.currentTurn + 1)
	g.board[cell] = mark

	if g.hasWon(cell, mark) {
		g.winner = mark
	} else if g.isBoardFull() {
		g.winner = 3 // draw
	} else {
		g.currentTurn = 1 - g.currentTurn
	}
	return nil
}

// hasWon checks whether the last mark placed at cell wins the game.
// It scans all 4 axis directions through the placed cell.
func (g *GameState) hasWon(cell int, mark int8) bool {
	row := cell / g.width
	col := cell % g.width
	dirs := [4][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		count := 1
		for _, sign := range []int{1, -1} {
			for i := 1; ; i++ {
				r := row + d[0]*sign*i
				c := col + d[1]*sign*i
				if r < 0 || r >= g.height || c < 0 || c >= g.width {
					break
				}
				if g.board[r*g.width+c] != mark {
					break
				}
				count++
			}
		}
		if count >= g.winLen {
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

// Board returns the board cells with player IDs (empty string for empty cells).
func (g *GameState) Board() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]string, len(g.board))
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

// Players returns both player IDs ([0] is the creator, [1] is the joiner).
func (g *GameState) Players() [2]string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.players
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
// Pass a zero-value GridConfig to use the default 3×3 board with win length 3.
func (w *Worker) CreateGame(playerID string, cfg GridConfig) *GameState {
	if cfg.Width == 0 || cfg.Height == 0 || cfg.WinLen == 0 {
		cfg = defaultGrid
	}
	id := fmt.Sprintf("game-%d-%d", w.id, w.nextID.Add(1))
	state := newGameState(id, playerID, cfg)
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


