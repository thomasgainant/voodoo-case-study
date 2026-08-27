package sharder_test

import (
	"fmt"
	"sync"
	"testing"

	"voodoo-case-study/internal/sharder"
)

func TestResolveIsDeterministic(t *testing.T) {
	s := sharder.New(4)
	gameID := "game-abc-123"

	first := s.Resolve(gameID)
	for i := 0; i < 10; i++ {
		if s.Resolve(gameID) != first {
			t.Fatal("Resolve returned a different Worker for the same gameId")
		}
	}
}

func TestResolveIndexCachesWorkerPointer(t *testing.T) {
	s := sharder.New(4)
	gameID := "game-xyz"

	w1 := s.Resolve(gameID)
	w2 := s.Resolve(gameID)
	if w1 != w2 {
		t.Error("expected identical Worker pointer from index cache on second call")
	}
}

func TestResolveDistributesAcrossAllWorkers(t *testing.T) {
	const numWorkers = 4
	s := sharder.New(numWorkers)

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
	s := sharder.New(8)
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

func TestSharderWorkersReturnsCorrectCount(t *testing.T) {
	const n = 8
	s := sharder.New(n)
	if len(s.Workers()) != n {
		t.Errorf("expected %d workers, got %d", n, len(s.Workers()))
	}
}
