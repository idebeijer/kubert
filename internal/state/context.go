package state

import (
	"errors"
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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	info, exists := m.state.Contexts[context]
	return info, exists
}

func (m *Manager) SetContextInfo(context string, info ContextInfo) error {
	m.state.Contexts[context] = info
	return m.Save()
}

func (m *Manager) RemoveContext(context string) error {
	delete(m.state.Contexts, context)
	return m.Save()
}

func (m *Manager) SetLastNamespace(context, namespace string) error {
	info, exists := m.state.Contexts[context]
	if !exists {
		return &ContextNotFoundError{Context: context}
	}
	info.LastNamespace = namespace
	m.state.Contexts[context] = info
	return m.Save()
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
	info, exists := m.state.Contexts[context]
	if !exists {
		return &ContextNotFoundError{Context: context}
	}
	info.Protected = &locked
	m.state.Contexts[context] = info
	return m.Save()
}

// DeleteContextProtection deletes the Protected field by setting it to nil
func (m *Manager) DeleteContextProtection(context string) error {
	info, exists := m.state.Contexts[context]
	if !exists {
		return &ContextNotFoundError{Context: context}
	}
	info.Protected = nil
	m.state.Contexts[context] = info
	return m.Save()
}

func (m *Manager) IsContextProtected(context string) (bool, error) {
	info, exists := m.state.Contexts[context]
	if !exists {
		return false, &ContextNotFoundError{Context: context}
	}
	if info.Protected == nil {
		return false, nil
	}
	return *info.Protected, nil
}

func (m *Manager) EnsureContextExists(context string) {
	if _, exists := m.state.Contexts[context]; !exists {
		m.state.Contexts[context] = ContextInfo{}
	}
}

// SetLastNamespaceWithContextCreation sets the last namespace for the given context. If the context does not exist, it will be created.
func (m *Manager) SetLastNamespaceWithContextCreation(context, namespace string) error {
	err := m.SetLastNamespace(context, namespace)
	if err != nil {
		var contextNotFoundError *ContextNotFoundError
		if errors.As(err, &contextNotFoundError) {
			m.EnsureContextExists(context)
			return m.SetLastNamespace(context, namespace)
		}
		return err
	}
	return nil
}
