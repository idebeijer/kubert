package state

import (
	"testing"

	"github.com/adrg/xdg"
)

func TestInPlaceSwitchWarnCount(t *testing.T) {
	xdg.DataHome = t.TempDir()

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	if got := m.InPlaceSwitchWarnCount(); got != 0 {
		t.Fatalf("initial count: want 0, got %d", got)
	}

	for i := 1; i <= InPlaceSwitchWarnMax; i++ {
		if err := m.RecordInPlaceSwitchWarn(); err != nil {
			t.Fatalf("RecordInPlaceSwitchWarn iteration %d: %v", i, err)
		}
		if got := m.InPlaceSwitchWarnCount(); got != i {
			t.Fatalf("after %d records: want %d, got %d", i, i, got)
		}
	}
}

func TestInPlaceSwitchWarnCount_Persisted(t *testing.T) {
	xdg.DataHome = t.TempDir()

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := m.RecordInPlaceSwitchWarn(); err != nil {
		t.Fatalf("RecordInPlaceSwitchWarn: %v", err)
	}

	// Re-open the state file and verify the count survived.
	m2, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager (2nd): %v", err)
	}
	if got := m2.InPlaceSwitchWarnCount(); got != 1 {
		t.Fatalf("persisted count: want 1, got %d", got)
	}
}
