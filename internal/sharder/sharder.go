package sharder

import (
	"hash/fnv"
	"sync"

	"voodoo-case-study/internal/worker"
)

// Sharder routes incoming gameIds to Workers using a hash index.
//
// Design mirrors the hash index pattern from database systems: a hash function
// maps each gameId to a bucket (Worker), and the index stores a direct pointer
// to that Worker in memory
//
// On first resolution, gameId is hashed to a Worker and the mapping is stored
// in the index. All subsequent calls hit the index in O(1) with a read lock,
// making it suitable for high-volume request routing.
type Sharder struct {
	workers []*worker.Worker
	// index is the hash index: gameId → *Worker (direct in-memory pointer).
	index   map[string]*worker.Worker
	pending map[string]struct{} // set of game IDs waiting for a second player
	mu      sync.RWMutex
}

func New(numWorkers int) *Sharder {
	workers := make([]*worker.Worker, numWorkers)
	for i := range workers {
		workers[i] = worker.New(i)
	}
	return &Sharder{
		workers: workers,
		index:   make(map[string]*worker.Worker),
		pending: make(map[string]struct{}),
	}
}

// Resolve returns the Worker responsible for gameID.
// New gameIDs are assigned via FNV-1a hash modulo the worker count and
// immediately cached in the index so subsequent calls skip hashing entirely.
func (s *Sharder) Resolve(gameID string) *worker.Worker {
	s.mu.RLock()
	w, ok := s.index[gameID]
	s.mu.RUnlock()
	if ok {
		return w
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Re-check after acquiring the write lock to handle concurrent first resolves.
	if w, ok = s.index[gameID]; ok {
		return w
	}
	w = s.workers[fnvHash(gameID)%uint32(len(s.workers))]
	s.index[gameID] = w
	return w
}

// Workers returns the list of active Workers.
func (s *Sharder) Workers() []*worker.Worker {
	return s.workers
}

// Register pins gameID to w in the index so that Resolve always routes to the correct worker.
// Must be called after CreateGame to ensure JoinGame and UpdateGame reach the same worker.
func (s *Sharder) Register(gameID string, w *worker.Worker) {
	s.mu.Lock()
	s.index[gameID] = w
	s.mu.Unlock()
}

// PickLeastLoaded returns the Worker with the fewest active game states.
func (s *Sharder) PickLeastLoaded() *worker.Worker {
	best := s.workers[0]
	for _, w := range s.workers[1:] {
		if w.StateCount() < best.StateCount() {
			best = w
		}
	}
	return best
}

// AddPending marks gameID as waiting for a second player.
func (s *Sharder) AddPending(gameID string) {
	s.mu.Lock()
	s.pending[gameID] = struct{}{}
	s.mu.Unlock()
}

// RemovePending removes gameID from the pending set when it is no longer waiting.
func (s *Sharder) RemovePending(gameID string) {
	s.mu.Lock()
	delete(s.pending, gameID)
	s.mu.Unlock()
}

// ListPending returns the IDs of all games currently waiting for a second player.
func (s *Sharder) ListPending() []string {
	s.mu.RLock()
	ids := make([]string, 0, len(s.pending))
	for id := range s.pending {
		ids = append(ids, id)
	}
	s.mu.RUnlock()
	return ids
}

func fnvHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
