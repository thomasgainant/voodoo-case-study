package sharder_test

import (
	"fmt"
	"sync"
	"testing"

	"voodoo-case-study/internal/sharder"
	"voodoo-case-study/internal/worker"
)

func TestResolveIsDeterministic(t *testing.T) {
	s := sharder.New(4, 1000)
	gameID := "game-abc-123"

	first := s.Resolve(gameID)
	for i := 0; i < 10; i++ {
		if s.Resolve(gameID) != first {
			t.Fatal("Resolve returned a different Worker for the same gameId")
		}
	}
}

func TestResolveIndexCachesWorkerPointer(t *testing.T) {
	s := sharder.New(4, 1000)
	gameID := "game-xyz"

	w1 := s.Resolve(gameID)
	w2 := s.Resolve(gameID)
	if w1 != w2 {
		t.Error("expected identical Worker pointer from index cache on second call")
	}
}

func TestResolveDistributesAcrossAllWorkers(t *testing.T) {
	const numWorkers = 4
	s := sharder.New(numWorkers, 1000)

	seen := make(map[int]bool)
	for i := 0; i < 100; i++ {
		w := s.Resolve(fmt.Sprintf("game-%d", i))
		seen[w.ID()] = true
	}
	if len(seen) != numWorkers {
		t.Errorf("expected all %d workers to receive at least one gameId, got %d", numWorkers, len(seen))
	}
}

func TestResolveIsThreadSafe(t *testing.T) {
	s := sharder.New(8, 1000)
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			gameID := fmt.Sprintf("game-%d", i%20) // overlap to stress cache writes
			s.Resolve(gameID)
		}(i)
	}
	wg.Wait()
}

func TestPickLeastLoadedReturnsWorkerWithFewestStates(t *testing.T) {
	s := sharder.New(3, 1000)
	workers := s.Workers()

	workers[0].CreateGame("p", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	workers[0].CreateGame("p", worker.GridConfig{Width: 3, Height: 3, WinLen: 3}) // 2 states
	workers[1].CreateGame("p", worker.GridConfig{Width: 3, Height: 3, WinLen: 3}) // 1 state
	// workers[2] has 0 states — must be picked

	got := s.PickLeastLoaded()
	if got != workers[2] {
		t.Errorf("expected workers[2] (0 states), got worker id=%d", got.ID())
	}
}

func TestPickLeastLoadedTieBreaksToFirstWorker(t *testing.T) {
	s := sharder.New(3, 1000)
	// All workers start empty — first one should be returned.
	got := s.PickLeastLoaded()
	if got != s.Workers()[0] {
		t.Errorf("expected workers[0] on empty tie, got worker id=%d", got.ID())
	}
}

func TestRegisterOverridesHashBasedRouting(t *testing.T) {
	s := sharder.New(4, 1000)
	w := s.PickLeastLoaded()
	state := w.CreateGame("p1", worker.GridConfig{Width: 3, Height: 3, WinLen: 3})
	s.Register(state.ID, w)

	resolved := s.Resolve(state.ID)
	if resolved != w {
		t.Errorf("expected registered worker (id=%d), got worker id=%d", w.ID(), resolved.ID())
	}
}

func TestSharderWorkersReturnsCorrectCount(t *testing.T) {
	const n = 8
	s := sharder.New(n, 1000)
	if len(s.Workers()) != n {
		t.Errorf("expected %d workers, got %d", n, len(s.Workers()))
	}
}

func TestAddPendingAppearsInList(t *testing.T) {
	s := sharder.New(2, 1000)
	s.AddPending("game-1")
	ids := s.ListPending()
	if len(ids) != 1 || ids[0] != "game-1" {
		t.Errorf("expected [game-1], got %v", ids)
	}
}

func TestRemovePendingDisappearsFromList(t *testing.T) {
	s := sharder.New(2, 1000)
	s.AddPending("game-1")
	s.RemovePending("game-1")
	if ids := s.ListPending(); len(ids) != 0 {
		t.Errorf("expected empty pending list, got %v", ids)
	}
}

func TestListPendingIsEmptyInitially(t *testing.T) {
	s := sharder.New(2, 1000)
	if ids := s.ListPending(); len(ids) != 0 {
		t.Errorf("expected empty pending list on new sharder, got %v", ids)
	}
}

func TestListPendingReturnsAllPendingGames(t *testing.T) {
	s := sharder.New(2, 1000)
	s.AddPending("game-a")
	s.AddPending("game-b")
	s.AddPending("game-c")
	ids := s.ListPending()
	if len(ids) != 3 {
		t.Errorf("expected 3 pending games, got %d: %v", len(ids), ids)
	}
}

func TestPickLeastLoadedSpawnsWorkerWhenOverloaded(t *testing.T) {
	const maxLoad = 2
	s := sharder.New(1, maxLoad)
	cfg := worker.GridConfig{Width: 3, Height: 3, WinLen: 3}

	// Fill the sole worker up to the threshold.
	for i := 0; i < maxLoad; i++ {
		w := s.PickLeastLoaded()
		w.CreateGame("p", cfg)
	}

	initialCount := len(s.Workers())
	if initialCount != 1 {
		t.Fatalf("expected 1 worker before threshold, got %d", initialCount)
	}

	// This call should trigger a spawn.
	s.PickLeastLoaded()

	if got := len(s.Workers()); got != 2 {
		t.Errorf("expected 2 workers after exceeding maxLoad, got %d", got)
	}
}

func TestPickLeastLoadedSpawnedWorkerReceivesNewGames(t *testing.T) {
	const maxLoad = 1
	s := sharder.New(1, maxLoad)
	cfg := worker.GridConfig{Width: 3, Height: 3, WinLen: 3}

	// Saturate the initial worker.
	w0 := s.PickLeastLoaded()
	w0.CreateGame("p", cfg)

	// Next pick must return a freshly spawned worker, not the saturated one.
	w1 := s.PickLeastLoaded()
	if w1 == w0 {
		t.Error("expected a new worker after saturation, got the same worker")
	}
	if w1.StateCount() != 0 {
		t.Errorf("expected spawned worker to start empty, got %d states", w1.StateCount())
	}
}
