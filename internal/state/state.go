package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
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
}

func NewManager() (*Manager, error) {
	dataDir := filepath.Join(xdg.DataHome, appName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	manager := &Manager{
		filename: filepath.Join(dataDir, stateFile),
		state: State{
			Contexts: make(map[string]ContextInfo),
		},
	}

	if err := manager.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return manager, nil
}

func (m *Manager) Load() error {
	data, err := os.ReadFile(m.filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m.state)
}

func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filename, data, 0644)
}
