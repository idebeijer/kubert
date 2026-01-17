package state

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/adrg/xdg"
)

const (
	testContextName   = "test-context"
	testNamespaceName = "test-namespace"
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
	_ = os.RemoveAll(tempDir)
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

	context := testContextName
	namespace := testNamespaceName

	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	info, exists := manager.ContextInfo(context)
	if !exists || info.LastNamespace != namespace {
		t.Errorf("ContextInfo() failed, got %v, want %v", info.LastNamespace, namespace)
	}

	_, exists = manager.ContextInfo("non-existing-context")
	if exists {
		t.Errorf("ContextInfo() should return false for non-existing context")
	}
}

func TestManager_RemoveContext(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := testContextName
	namespace := testNamespaceName

	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	if err := manager.RemoveContext(context); err != nil {
		t.Fatal(err)
	}

	_, exists := manager.ContextInfo(context)
	if exists {
		t.Errorf("RemoveContext() failed, context still exists")
	}
}

func TestManager_ListContexts(t *testing.T) {
	tests := []struct {
		name     string
		contexts []string
	}{
		{
			name:     "empty",
			contexts: []string{},
		},
		{
			name:     "single context",
			contexts: []string{"context1"},
		},
		{
			name:     "multiple contexts",
			contexts: []string{"context1", "context2", "context3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, tempDir := setupTestManager(t)
			defer cleanupTestManager(tempDir)

			namespace := testNamespaceName
			for _, context := range tt.contexts {
				if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
					t.Fatal(err)
				}
			}

			listedContexts := manager.ListContexts()
			if len(listedContexts) != len(tt.contexts) {
				t.Errorf("ListContexts() failed, expected %d contexts, got %d", len(tt.contexts), len(listedContexts))
			}

			for _, context := range tt.contexts {
				found := slices.Contains(listedContexts, context)
				if !found {
					t.Errorf("ListContexts() missing context %v", context)
				}
			}
		})
	}
}

func TestStateManager_PersistenceAcrossInstances(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := testContextName
	namespace := testNamespaceName

	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	newManager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	info, exists := newManager.ContextInfo(context)
	if !exists || info.LastNamespace != namespace {
		t.Errorf("Persistence across instances failed, got %v, want %v", info.LastNamespace, namespace)
	}
}

func TestManager_ContextProtection(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := testContextName
	namespace := testNamespaceName

	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	if err := manager.SetContextProtection(context, true); err != nil {
		t.Fatal(err)
	}

	protected, err := manager.IsContextProtected(context)
	if err != nil {
		t.Fatal(err)
	}
	if !protected {
		t.Errorf("IsContextProtected() failed, expected true, got %v", protected)
	}

	if err := manager.DeleteContextProtection(context); err != nil {
		t.Fatal(err)
	}

	protected, err = manager.IsContextProtected(context)
	if err != nil {
		t.Fatal(err)
	}
	if protected {
		t.Errorf("IsContextProtected() failed, expected false, got %v", protected)
	}

	_, err = manager.IsContextProtected("non-existing")
	if err == nil {
		t.Errorf("IsContextProtected() should return error for non-existing context")
	}
	var contextNotFoundError *ContextNotFoundError
	if !errors.As(err, &contextNotFoundError) {
		t.Errorf("IsContextProtected() should return ContextNotFoundError, got %T", err)
	}
}

func TestManager_SetLastNamespace(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := testContextName
	namespace := testNamespaceName

	err := manager.SetLastNamespace(context, namespace)
	if err == nil {
		t.Errorf("SetLastNamespace() should return error for non-existing context")
	}
	var contextNotFoundError *ContextNotFoundError
	if !errors.As(err, &contextNotFoundError) {
		t.Errorf("SetLastNamespace() should return ContextNotFoundError, got %T", err)
	}

	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	newNamespace := "updated-namespace"
	if err := manager.SetLastNamespace(context, newNamespace); err != nil {
		t.Fatal(err)
	}

	info, exists := manager.ContextInfo(context)
	if !exists || info.LastNamespace != newNamespace {
		t.Errorf("SetLastNamespace() failed, got %v, want %v", info.LastNamespace, newNamespace)
	}
}

func TestManager_EnsureContextExists(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := "test-context"

	_, exists := manager.ContextInfo(context)
	if exists {
		t.Errorf("Context should not exist initially")
	}

	if err := manager.EnsureContextExists(context); err != nil {
		t.Fatal(err)
	}

	info, exists := manager.ContextInfo(context)
	if !exists {
		t.Errorf("EnsureContextExists() failed, context should exist")
	}
	if info.LastNamespace != "" {
		t.Errorf("EnsureContextExists() should create empty context, got %v", info.LastNamespace)
	}

	if err := manager.EnsureContextExists(context); err != nil {
		t.Fatal(err)
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	const numGoroutines = 10
	const numOperations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range numOperations {
				context := fmt.Sprintf("context-%d-%d", id, j)
				namespace := fmt.Sprintf("namespace-%d-%d", id, j)

				if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
					t.Errorf("Concurrent SetLastNamespaceWithContextCreation failed: %v", err)
					return
				}

				info, exists := manager.ContextInfo(context)
				if !exists {
					t.Errorf("Concurrent ContextInfo failed: context %s not found", context)
					return
				}
				if info.LastNamespace != namespace {
					t.Errorf("Concurrent ContextInfo failed: expected %s, got %s", namespace, info.LastNamespace)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	contexts := manager.ListContexts()
	expectedCount := numGoroutines * numOperations
	if len(contexts) != expectedCount {
		t.Errorf("Concurrent test failed: expected %d contexts, got %d", expectedCount, len(contexts))
	}
}

func TestManager_ErrorHandling(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	nonExistingContext := "non-existing-context"

	tests := []struct {
		name      string
		operation func() error
	}{
		{
			name:      "SetLastNamespace",
			operation: func() error { return manager.SetLastNamespace(nonExistingContext, "namespace") },
		},
		{
			name:      "SetContextProtection",
			operation: func() error { return manager.SetContextProtection(nonExistingContext, true) },
		},
		{
			name:      "DeleteContextProtection",
			operation: func() error { return manager.DeleteContextProtection(nonExistingContext) },
		},
		{
			name: "IsContextProtected",
			operation: func() error {
				_, err := manager.IsContextProtected(nonExistingContext)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			if err == nil {
				t.Errorf("%s should fail for non-existing context", tt.name)
			}

			var contextNotFoundError *ContextNotFoundError
			if !errors.As(err, &contextNotFoundError) {
				t.Errorf("Expected ContextNotFoundError, got %T", err)
			}
		})
	}
}

func TestManager_LiftContextProtection(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := testContextName
	namespace := testNamespaceName

	// Create context first
	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}

	// Set protection
	if err := manager.SetContextProtection(context, true); err != nil {
		t.Fatal(err)
	}

	// Verify it's protected
	protected, err := manager.IsContextProtected(context)
	if err != nil {
		t.Fatal(err)
	}
	if !protected {
		t.Error("Expected context to be protected before lift")
	}

	// Lift protection for 1 hour
	liftUntil := time.Now().Add(1 * time.Hour)
	if err := manager.LiftContextProtection(context, liftUntil); err != nil {
		t.Fatal(err)
	}

	// Verify protection is lifted (should return false because lift is active)
	protected, err = manager.IsContextProtected(context)
	if err != nil {
		t.Fatal(err)
	}
	if protected {
		t.Error("Expected context protection to be lifted")
	}

	// Verify ProtectedUntil is set correctly
	info, exists := manager.ContextInfo(context)
	if !exists {
		t.Fatal("Context should exist")
	}
	if info.ProtectedUntil == nil {
		t.Error("ProtectedUntil should be set")
	}
	if !info.ProtectedUntil.Equal(liftUntil) {
		t.Errorf("ProtectedUntil = %v, want %v", info.ProtectedUntil, liftUntil)
	}
}

func TestManager_LiftContextProtection_Expired(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := testContextName
	namespace := testNamespaceName

	// Create context and set protection
	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetContextProtection(context, true); err != nil {
		t.Fatal(err)
	}

	// Lift protection with an already-expired time
	expiredTime := time.Now().Add(-1 * time.Hour)
	if err := manager.LiftContextProtection(context, expiredTime); err != nil {
		t.Fatal(err)
	}

	// Verify protection is still active (lift expired)
	protected, err := manager.IsContextProtected(context)
	if err != nil {
		t.Fatal(err)
	}
	if !protected {
		t.Error("Expected context to still be protected after expired lift")
	}
}

func TestManager_ClearProtectedUntil(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	context := testContextName
	namespace := testNamespaceName

	// Create context and set protection with lift
	if err := manager.SetLastNamespaceWithContextCreation(context, namespace); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetContextProtection(context, true); err != nil {
		t.Fatal(err)
	}
	liftUntil := time.Now().Add(1 * time.Hour)
	if err := manager.LiftContextProtection(context, liftUntil); err != nil {
		t.Fatal(err)
	}

	// Verify lift is active
	protected, _ := manager.IsContextProtected(context)
	if protected {
		t.Error("Expected lift to be active")
	}

	// Clear the lift
	if err := manager.ClearProtectedUntil(context); err != nil {
		t.Fatal(err)
	}

	// Verify protection is restored
	protected, err := manager.IsContextProtected(context)
	if err != nil {
		t.Fatal(err)
	}
	if !protected {
		t.Error("Expected context to be protected after clearing lift")
	}

	// Verify ProtectedUntil is nil
	info, _ := manager.ContextInfo(context)
	if info.ProtectedUntil != nil {
		t.Error("ProtectedUntil should be nil after clear")
	}
}

func TestManager_LiftContextProtection_NonExistingContext(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	err := manager.LiftContextProtection("non-existing", time.Now().Add(1*time.Hour))
	if err == nil {
		t.Error("LiftContextProtection should fail for non-existing context")
	}

	var contextNotFoundError *ContextNotFoundError
	if !errors.As(err, &contextNotFoundError) {
		t.Errorf("Expected ContextNotFoundError, got %T", err)
	}
}

func TestManager_ClearProtectedUntil_NonExistingContext(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	err := manager.ClearProtectedUntil("non-existing")
	if err == nil {
		t.Error("ClearProtectedUntil should fail for non-existing context")
	}

	var contextNotFoundError *ContextNotFoundError
	if !errors.As(err, &contextNotFoundError) {
		t.Errorf("Expected ContextNotFoundError, got %T", err)
	}
}
