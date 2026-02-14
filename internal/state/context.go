package state

import (
	"fmt"
	"time"
)

type ContextNotFoundError struct {
	Context string
}

func (e *ContextNotFoundError) Error() string {
	return fmt.Sprintf("context '%s' not found", e.Context)
}

type ContextInfo struct {
	LastNamespace  string     `json:"last_namespace"`
	Protected      *bool      `json:"protected,omitempty"`
	ProtectedUntil *time.Time `json:"protected_until,omitempty"`
}

func (m *Manager) ContextInfo(context string) (ContextInfo, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	info, exists := m.state.Contexts[context]
	return info, exists
}

func (m *Manager) SetContextInfo(context string, info ContextInfo) error {
	return m.withLock(func() error {
		m.state.Contexts[context] = info
		return m.saveState()
	})
}

func (m *Manager) RemoveContext(context string) error {
	return m.withLock(func() error {
		delete(m.state.Contexts, context)
		return m.saveState()
	})
}

func (m *Manager) SetLastNamespace(context, namespace string) error {
	return m.withLock(func() error {
		info, exists := m.state.Contexts[context]
		if !exists {
			return &ContextNotFoundError{Context: context}
		}
		info.LastNamespace = namespace
		m.state.Contexts[context] = info
		return m.saveState()
	})
}

func (m *Manager) ListContexts() []string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	contexts := make([]string, 0, len(m.state.Contexts))
	for context := range m.state.Contexts {
		contexts = append(contexts, context)
	}
	return contexts
}

func (m *Manager) SetContextProtection(context string, locked bool) error {
	return m.withLock(func() error {
		info, exists := m.state.Contexts[context]
		if !exists {
			return &ContextNotFoundError{Context: context}
		}
		info.Protected = &locked
		m.state.Contexts[context] = info
		return m.saveState()
	})
}

func (m *Manager) DeleteContextProtection(context string) error {
	return m.withLock(func() error {
		info, exists := m.state.Contexts[context]
		if !exists {
			return &ContextNotFoundError{Context: context}
		}
		info.Protected = nil
		m.state.Contexts[context] = info
		return m.saveState()
	})
}

func (m *Manager) IsContextProtected(context string) (bool, error) {
	m.mutex.Lock()
	info, exists := m.state.Contexts[context]
	m.mutex.Unlock()

	if !exists {
		return false, &ContextNotFoundError{Context: context}
	}

	// Check if protection is temporarily lifted
	if info.ProtectedUntil != nil {
		if time.Now().Before(*info.ProtectedUntil) {
			return false, nil
		}
		// Lift has expired, clean up (best effort)
		_ = m.ClearProtectedUntil(context)
	}

	if info.Protected == nil {
		return false, nil
	}
	return *info.Protected, nil
}

// LiftContextProtection temporarily lifts protection for the given context until the specified time
func (m *Manager) LiftContextProtection(context string, until time.Time) error {
	return m.withLock(func() error {
		info, exists := m.state.Contexts[context]
		if !exists {
			return &ContextNotFoundError{Context: context}
		}
		info.ProtectedUntil = &until
		m.state.Contexts[context] = info
		return m.saveState()
	})
}

// ClearProtectedUntil clears the ProtectedUntil field for a context
func (m *Manager) ClearProtectedUntil(context string) error {
	return m.withLock(func() error {
		info, exists := m.state.Contexts[context]
		if !exists {
			return &ContextNotFoundError{Context: context}
		}
		info.ProtectedUntil = nil
		m.state.Contexts[context] = info
		return m.saveState()
	})
}

func (m *Manager) EnsureContextExists(context string) error {
	return m.withLock(func() error {
		if _, exists := m.state.Contexts[context]; !exists {
			m.state.Contexts[context] = ContextInfo{}
			return m.saveState()
		}
		return nil
	})
}

func (m *Manager) SetLastNamespaceWithContextCreation(context, namespace string) error {
	return m.withLock(func() error {
		info, exists := m.state.Contexts[context]
		if !exists {
			m.state.Contexts[context] = ContextInfo{LastNamespace: namespace}
		} else {
			info.LastNamespace = namespace
			m.state.Contexts[context] = info
		}
		return m.saveState()
	})
}

func (m *Manager) GetLastContext() (string, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.state.LastContext, m.state.LastContext != ""
}

func (m *Manager) SetLastContext(context string) error {
	return m.withLock(func() error {
		m.state.LastContext = context
		return m.saveState()
	})
}
