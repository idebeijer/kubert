package state

import (
	"encoding/json"
	"fmt"
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

type State struct {
	Contexts map[string]ContextInfo `json:"contexts"`
}

type Manager struct {
	filename string
	state    State
	fileLock *flock.Flock
	mutex    sync.Mutex
}

func NewManager() (*Manager, error) {
	dataDir := filepath.Join(xdg.DataHome, appName)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
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

	// Check if the state file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// Create the state file with an initial empty state
		if err := manager.Save(); err != nil {
			return nil, err
		}
	} else {
		if err := manager.Load(); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

func FilePath() (string, error) {
	return filepath.Join(xdg.DataHome, appName, stateFile), nil
}

func (m *Manager) withLock(fn func() error) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.fileLock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer func() {
		if unlockErr := m.fileLock.Unlock(); unlockErr != nil {
			// Log the error but don't override the original error
		}
	}()

	return fn()
}

func (m *Manager) withMemoryLock(fn func() error) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return fn()
}

func (m *Manager) Lock() error {
	m.mutex.Lock()
	if err := m.fileLock.Lock(); err != nil {
		m.mutex.Unlock()
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	return nil
}

func (m *Manager) Unlock() error {
	defer m.mutex.Unlock()
	if err := m.fileLock.Unlock(); err != nil {
		return fmt.Errorf("failed to release file lock: %w", err)
	}
	return nil
}

func (m *Manager) Load() error {
	return m.withLock(func() error {
		data, err := os.ReadFile(m.filename)
		if err != nil {
			return fmt.Errorf("failed to read state file: %w", err)
		}

		return json.Unmarshal(data, &m.state)
	})
}

func (m *Manager) Save() error {
	return m.withLock(func() error {
		return m.saveState()
	})
}

// saveState saves the current state without acquiring locks (internal use only)
func (m *Manager) saveState() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return os.WriteFile(m.filename, data, 0o644)
}
