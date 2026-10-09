package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	if _, ok := errors.AsType[*ContextNotFoundError](err); !ok {
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
	if _, ok := errors.AsType[*ContextNotFoundError](err); !ok {
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

			if _, ok := errors.AsType[*ContextNotFoundError](err); !ok {
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

	if _, ok := errors.AsType[*ContextNotFoundError](err); !ok {
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

	if _, ok := errors.AsType[*ContextNotFoundError](err); !ok {
		t.Errorf("Expected ContextNotFoundError, got %T", err)
	}
}

// Two Managers over one state file stand in for two kubert processes. Each
// mutation writes the whole struct back, so a stale snapshot loses the other's
// change.
func TestManager_ConcurrentProcessesDoNotLoseUpdates(t *testing.T) {
	_, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	seed, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	for _, context := range []string{"ctx-a", "ctx-b"} {
		if err := seed.SetLastNamespaceWithContextCreation(context, "original"); err != nil {
			t.Fatal(err)
		}
	}

	// Both processes start up and read the same snapshot.
	first, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	if err := first.SetLastNamespace("ctx-a", "from-first"); err != nil {
		t.Fatal(err)
	}
	if err := second.SetLastNamespace("ctx-b", "from-second"); err != nil {
		t.Fatal(err)
	}

	final, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	for context, want := range map[string]string{"ctx-a": "from-first", "ctx-b": "from-second"} {
		info, exists := final.ContextInfo(context)
		if !exists {
			t.Fatalf("context %s missing from state file", context)
		}
		if info.LastNamespace != want {
			t.Errorf("context %s: expected namespace %q, got %q", context, want, info.LastNamespace)
		}
	}
}

// A removed context must not come back when another process writes its stale
// snapshot.
func TestManager_ConcurrentProcessesDoNotResurrectRemovedContext(t *testing.T) {
	_, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	seed, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	for _, context := range []string{"doomed", "keeper"} {
		if err := seed.SetLastNamespaceWithContextCreation(context, "ns"); err != nil {
			t.Fatal(err)
		}
	}

	remover, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	writer, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	if err := remover.RemoveContext("doomed"); err != nil {
		t.Fatal(err)
	}
	// writer still has "doomed" in its snapshot from startup.
	if err := writer.SetLastNamespace("keeper", "updated"); err != nil {
		t.Fatal(err)
	}

	final, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := final.ContextInfo("doomed"); exists {
		t.Error("removed context was resurrected by a stale writer")
	}
	if info, _ := final.ContextInfo("keeper"); info.LastNamespace != "updated" {
		t.Errorf("expected keeper namespace %q, got %q", "updated", info.LastNamespace)
	}
}

// Clearing an expired lift seen in a stale snapshot must not wipe a fresh lift
// another process set since.
func TestManager_ExpiredLiftCleanupKeepsFreshLift(t *testing.T) {
	_, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	seed, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.SetLastNamespaceWithContextCreation(testContextName, testNamespaceName); err != nil {
		t.Fatal(err)
	}
	if err := seed.SetContextProtection(testContextName, true); err != nil {
		t.Fatal(err)
	}
	if err := seed.LiftContextProtection(testContextName, time.Now().Add(-1*time.Hour)); err != nil {
		t.Fatal(err)
	}

	// checker starts up and sees the expired lift.
	checker, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	lifter, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	freshLift := time.Now().Add(1 * time.Hour)
	if err := lifter.LiftContextProtection(testContextName, freshLift); err != nil {
		t.Fatal(err)
	}

	if _, err := checker.IsContextProtected(testContextName); err != nil {
		t.Fatal(err)
	}

	final, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	info, _ := final.ContextInfo(testContextName)
	if info.ProtectedUntil == nil {
		t.Fatal("fresh lift was wiped by a stale expired-lift cleanup")
	}
	if !info.ProtectedUntil.Equal(freshLift) {
		t.Errorf("ProtectedUntil = %v, want %v", info.ProtectedUntil, freshLift)
	}
}

// Deleting state.json must reset the state rather than let the next mutator
// rewrite the snapshot from before the delete.
func TestManager_DeletedStateFileResetsState(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	if err := manager.SetLastNamespaceWithContextCreation("doomed", "ns"); err != nil {
		t.Fatal(err)
	}

	stateFilePath, err := FilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(stateFilePath); err != nil {
		t.Fatal(err)
	}

	if err := manager.SetLastNamespaceWithContextCreation("fresh", "ns"); err != nil {
		t.Fatal(err)
	}

	if contexts := manager.ListContexts(); !slices.Equal(contexts, []string{"fresh"}) {
		t.Errorf("expected only the post-delete context, got %v", contexts)
	}
}

// saveState writes through a temp file, so the 0600 mode os.WriteFile once set
// explicitly is now implicit in os.CreateTemp. Pins the mode; the leftover check
// only covers the happy path, since the error paths never run here.
func TestManager_SaveStateLeavesNoTempFilesAndKeepsMode(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	if err := manager.SetLastNamespaceWithContextCreation(testContextName, testNamespaceName); err != nil {
		t.Fatal(err)
	}

	stateFilePath, err := FilePath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(stateFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("expected state file mode 0600, got %#o", mode)
	}

	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(stateFilePath), "*.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) > 0 {
		t.Errorf("temporary files left behind: %v", leftovers)
	}
}

// A failed release leaves gofrs/flock believing it still holds the lock, so the
// Manager refuses further work rather than running with no exclusion at all. The
// flag is set directly here; the syscall failure itself isn't reproducible.
func TestManager_RefusesWorkAfterFailedUnlock(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestManager(tempDir)

	manager.lockBroken = true

	// Twice: the second call also proves lock() released the mutex on refusal.
	for range 2 {
		err := manager.SetLastNamespaceWithContextCreation(testContextName, testNamespaceName)
		if !errors.Is(err, errLockBroken) {
			t.Fatalf("expected errLockBroken, got %v", err)
		}
	}
}
