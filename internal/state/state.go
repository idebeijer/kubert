package state

import (
	"encoding/json"
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
	if err := os.MkdirAll(dataDir, 0755); err != nil {
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

func (m *Manager) Lock() error {
	m.mutex.Lock()
	return m.fileLock.Lock()
}

func (m *Manager) Unlock() error {
	err := m.fileLock.Unlock()
	m.mutex.Unlock()
	return err
}

func (m *Manager) Load() error {
	if err := m.Lock(); err != nil {
		return err
	}
	defer m.Unlock()

	data, err := os.ReadFile(m.filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m.state)
}

func (m *Manager) Save() error {
	if err := m.Lock(); err != nil {
		return err
	}
	defer m.Unlock()

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filename, data, 0644)
}
