package worker_test

import (
	"testing"

	"voodoo-case-study/internal/worker"
)

func TestWorkerID(t *testing.T) {
	w := worker.New(3)
	if w.ID() != 3 {
		t.Errorf("expected id=3, got %d", w.ID())
	}
}
