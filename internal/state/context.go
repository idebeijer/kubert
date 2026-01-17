package state

import (
	"fmt"
)

type ContextNotFoundError struct {
	Context string
}

func (e *ContextNotFoundError) Error() string {
	return fmt.Sprintf("context '%s' not found", e.Context)
}

type ContextInfo struct {
	LastNamespace string `json:"last_namespace"`
	Protected     *bool  `json:"protected,omitempty"`
}

func (m *Manager) ContextInfo(context string) (ContextInfo, bool) {
	var info ContextInfo
	var exists bool

	_ = m.withMemoryLock(func() error {
		info, exists = m.state.Contexts[context]
		return nil
	})

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
	var contexts []string

	_ = m.withMemoryLock(func() error {
		contexts = make([]string, 0, len(m.state.Contexts))
		for context := range m.state.Contexts {
			contexts = append(contexts, context)
		}
		return nil
	})

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
	var result bool
	var err error

	_ = m.withMemoryLock(func() error {
		info, exists := m.state.Contexts[context]
		if !exists {
			err = &ContextNotFoundError{Context: context}
			return nil
		}
		if info.Protected == nil {
			result = false
		} else {
			result = *info.Protected
		}
		return nil
	})

	return result, err
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
	var lastContext string
	var exists bool

	_ = m.withMemoryLock(func() error {
		lastContext = m.state.LastContext
		exists = lastContext != ""
		return nil
	})

	return lastContext, exists
}

func (m *Manager) SetLastContext(context string) error {
	return m.withLock(func() error {
		m.state.LastContext = context
		return m.saveState()
	})
}
