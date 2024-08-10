package state

import (
	"os"
	"testing"

	"github.com/adrg/xdg"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	tempDir, err := os.MkdirTemp("", "kubert_test")
	if err != nil {
		t.Fatal(err)
	}

	xdg.DataHome = tempDir

	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	return manager, tempDir
}

func cleanupTestManager(tempDir string) {
	os.RemoveAll(tempDir)
}

func TestManager_SetLastNamespaceWithContextCreation(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := "context"
	namespace := "namespace"

	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	info, exists := manager.ContextInfo(context)
	if !exists {
		t.Errorf("SetLastNamespaceWithContextCreation() failed, got %v, want %v", info.LastNamespace, namespace)
	}

	newNamespace := "new-namespace"
	if err := manager.SetLastNamespaceWithContextCreation(context, newNamespace); err != nil {
		t.Fatal(err)
	}

	info, exists = manager.ContextInfo(context)
	if !exists {
		t.Errorf("SetLastNamespaceWithContextCreation() failed, got %v, want %v", info.LastNamespace, newNamespace)
	}
}

func TestManager_ContextInfo(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := "test-context"
	namespace := "test-namespace"

	// Add context
	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	// Verify context info
	info, exists := manager.ContextInfo(context)
	if !exists || info.LastNamespace != namespace {
		t.Errorf("ContextInfo() failed, got %v, want %v", info.LastNamespace, namespace)
	}

	// Verify non-existing context
	_, exists = manager.ContextInfo("non-existing-context")
	if exists {
		t.Errorf("ContextInfo() should return false for non-existing context")
	}
}

func TestManager_RemoveContext(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := "test-context"
	namespace := "test-namespace"

	// Add context
	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	// Remove context
	if err := manager.RemoveContext(context); err != nil {
		t.Fatal(err)
	}

	// Verify context is removed
	_, exists := manager.ContextInfo(context)
	if exists {
		t.Errorf("RemoveContext() failed, context still exists")
	}
}

func TestManager_ListContexts(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	contexts := []string{"context1", "context2", "context3"}
	namespace := "test-namespace"

	// Add multiple contexts
	for _, context := range contexts {
		if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
			t.Fatal(err)
		}
	}

	// Verify all contexts are listed
	listedContexts := manager.ListContexts()
	if len(listedContexts) != len(contexts) {
		t.Errorf("ListContexts() failed, got %v, want %v", listedContexts, contexts)
	}

	for _, context := range contexts {
		found := false
		for _, listedContext := range listedContexts {
			if context == listedContext {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListContexts() missing context %v", context)
		}
	}
}

func TestStateManager_PersistenceAcrossInstances(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := "test-context"
	namespace := "test-namespace"

	// Add context
	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	// Create a new manager
	newManager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	// Verify context info
	info, exists := newManager.ContextInfo(context)
	if !exists || info.LastNamespace != namespace {
		t.Errorf("Persistence across instances failed, got %v, want %v", info.LastNamespace, namespace)
	}
}
