package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/adrg/xdg"
	"github.com/gofrs/flock"
)

const (
	appName   = "kubert"
	stateFile = "state.json"
)

var errLockBroken = errors.New("file lock is in an unknown state after a failed release; rerun the command")

type State struct {
	Contexts               map[string]ContextInfo `json:"contexts"`
	LastContext            string                 `json:"last_context,omitempty"`
	InPlaceSwitchWarnCount int                    `json:"in_place_switch_warn_count,omitempty"`
}

// Manager holds the state file's contents in memory. Mutations reload under the
// file lock so they never overwrite another process's changes; reads are served
// from the snapshot taken at the last mutation, or at NewManager.
type Manager struct {
	filename   string
	state      State
	fileLock   *flock.Flock
	mutex      sync.Mutex
	lockBroken bool
}

func NewManager() (*Manager, error) {
	dataDir := filepath.Join(xdg.DataHome, appName)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}

	fullPath := filepath.Join(dataDir, stateFile)
	manager := &Manager{
		filename: fullPath,
		state: State{
			Contexts: make(map[string]ContextInfo),
		},
		fileLock: flock.New(fullPath + ".lock"),
	}

	// Acquire lock before checking/creating state file to avoid race conditions
	if err := manager.lock(); err != nil {
		return nil, fmt.Errorf("failed to acquire lock during initialization: %w", err)
	}
	defer func() {
		if unlockErr := manager.unlock(); unlockErr != nil {
			slog.Warn("failed to release lock during initialization", "error", unlockErr)
		}
	}()

	if _, err := os.Stat(fullPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to stat state file: %w", err)
		}
		if err := manager.saveState(); err != nil {
			return nil, err
		}
	} else if err := manager.reload(); err != nil {
		return nil, err
	}

	return manager, nil
}

func FilePath() (string, error) {
	return filepath.Join(xdg.DataHome, appName, stateFile), nil
}

func (m *Manager) withLock(fn func() error) error {
	if err := m.lock(); err != nil {
		return err
	}
	defer func() {
		if unlockErr := m.unlock(); unlockErr != nil {
			slog.Warn("failed to release file lock", "error", unlockErr)
		}
	}()

	// saveState writes the whole struct back, so a stale snapshot would discard
	// another process's changes. Refresh to keep read-modify-write under the lock.
	if err := m.reload(); err != nil {
		return err
	}

	return fn()
}

// reload replaces the in-memory state with the state file's contents; callers hold
// both the mutex and the file lock. Replacing matters rather than unmarshalling
// over m.state: json.Unmarshal merges into an existing map, so a context another
// process removed would come back. A deleted file resets for the same reason.
func (m *Manager) reload() error {
	var state State

	data, err := os.ReadFile(m.filename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read state file: %w", err)
	}
	if err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}
	if state.Contexts == nil {
		state.Contexts = make(map[string]ContextInfo)
	}

	m.state = state
	return nil
}

// lock holds both the mutex and the file lock until unlock. Unexported because
// sync.Mutex is not reentrant: a caller holding it deadlocks on the next Manager
// method it touches. Use withLock, which cannot be left holding the mutex.
func (m *Manager) lock() error {
	m.mutex.Lock()
	if m.lockBroken {
		m.mutex.Unlock()
		return errLockBroken
	}
	if err := m.fileLock.Lock(); err != nil {
		m.mutex.Unlock()
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	return nil
}

func (m *Manager) unlock() error {
	defer m.mutex.Unlock()
	if err := m.fileLock.Unlock(); err != nil {
		// flock leaves its "locked" flag set when the unlock syscall fails, and its
		// Lock returns nil without a syscall while that flag is set. Every later
		// lock would then hold nothing, so refuse the Manager instead of silently
		// dropping cross-process exclusion.
		m.lockBroken = true
		return fmt.Errorf("failed to release file lock: %w", err)
	}
	return nil
}

// saveState writes the in-memory state to the state file; callers hold both the
// mutex and the file lock.
func (m *Manager) saveState() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Same directory as the state file, so the rename stays on one filesystem.
	tmp, err := os.CreateTemp(filepath.Dir(m.filename), filepath.Base(m.filename)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary state file: %w", err)
	}

	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// Sync first: state.json must not end up pointing at unpersisted contents.
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to sync temporary state file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temporary state file: %w", err)
	}

	if err := os.Rename(tmpName, m.filename); err != nil {
		return fmt.Errorf("failed to replace state file: %w", err)
	}
	tmpName = "" // disarms the deferred remove

	return nil
}
