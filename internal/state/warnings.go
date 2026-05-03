package state

const InPlaceSwitchWarnMax = 3

// InPlaceSwitchWarnCount returns how many times the in-place switch notice has
// been shown to the user.
func (m *Manager) InPlaceSwitchWarnCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.state.InPlaceSwitchWarnCount
}

// RecordInPlaceSwitchWarn increments the notice counter and persists it.
func (m *Manager) RecordInPlaceSwitchWarn() error {
	return m.withLock(func() error {
		m.state.InPlaceSwitchWarnCount++
		return m.saveState()
	})
}
