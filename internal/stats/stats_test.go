package stats_test

import (
	"testing"

	"voodoo-case-study/internal/stats"
)

func TestGetReturnsZeroForUnknownPlayer(t *testing.T) {
	s := stats.New()
	r := s.Get("unknown")
	if r.Wins != 0 || r.Losses != 0 || r.Draws != 0 {
		t.Errorf("expected zero record, got %+v", r)
	}
}

func TestRecordWinIncrementsWins(t *testing.T) {
	s := stats.New()
	s.RecordWin("p1")
	s.RecordWin("p1")
	r := s.Get("p1")
	if r.Wins != 2 {
		t.Errorf("expected Wins=2, got %d", r.Wins)
	}
	if r.Losses != 0 || r.Draws != 0 {
		t.Errorf("expected Losses=0 Draws=0, got %+v", r)
	}
}

func TestRecordLossIncrementsLosses(t *testing.T) {
	s := stats.New()
	s.RecordLoss("p1")
	r := s.Get("p1")
	if r.Losses != 1 {
		t.Errorf("expected Losses=1, got %d", r.Losses)
	}
}

func TestRecordDrawIncrementsDraws(t *testing.T) {
	s := stats.New()
	s.RecordDraw("p1")
	r := s.Get("p1")
	if r.Draws != 1 {
		t.Errorf("expected Draws=1, got %d", r.Draws)
	}
}

func TestIndependentPlayersHaveSeparateRecords(t *testing.T) {
	s := stats.New()
	s.RecordWin("p1")
	s.RecordLoss("p2")
	if s.Get("p1").Wins != 1 || s.Get("p1").Losses != 0 {
		t.Errorf("p1 record wrong: %+v", s.Get("p1"))
	}
	if s.Get("p2").Wins != 0 || s.Get("p2").Losses != 1 {
		t.Errorf("p2 record wrong: %+v", s.Get("p2"))
	}
}
