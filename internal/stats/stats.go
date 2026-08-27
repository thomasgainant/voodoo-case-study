package stats

import "sync"

// Record holds the outcome counts for one player.
type Record struct {
	Wins, Losses, Draws int64
}

// Store tracks win/loss/draw counts for all players.
type Store struct {
	mu      sync.RWMutex
	records map[string]*Record
}

func New() *Store {
	return &Store{records: make(map[string]*Record)}
}

func (s *Store) getOrCreate(playerID string) *Record {
	r, ok := s.records[playerID]
	if !ok {
		r = &Record{}
		s.records[playerID] = r
	}
	return r
}

func (s *Store) RecordWin(playerID string) {
	s.mu.Lock()
	s.getOrCreate(playerID).Wins++
	s.mu.Unlock()
}

func (s *Store) RecordLoss(playerID string) {
	s.mu.Lock()
	s.getOrCreate(playerID).Losses++
	s.mu.Unlock()
}

func (s *Store) RecordDraw(playerID string) {
	s.mu.Lock()
	s.getOrCreate(playerID).Draws++
	s.mu.Unlock()
}

// Get returns a snapshot of the player's record (zero value if never played).
func (s *Store) Get(playerID string) Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.records[playerID]; ok {
		return *r
	}
	return Record{}
}
