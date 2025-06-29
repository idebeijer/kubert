package kubeconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/idebeijer/kubert/internal/keychain"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// EncryptedContext represents metadata about an encrypted kubeconfig context
type EncryptedContext struct {
	Name          string `json:"name"`
	EncryptedFile string `json:"encryptedFile"`
	OriginalFile  string `json:"originalFile,omitempty"`
}

// MacOSEncryptedProvider provides encrypted kubeconfig storage using macOS Keychain
type MacOSEncryptedProvider struct {
	StorageDir      string
	ContextsFile    string
	keychainManager *keychain.Manager
}

// NewMacOSEncryptedProvider creates a new encrypted provider
func NewMacOSEncryptedProvider(storageDir string) (*MacOSEncryptedProvider, error) {
	if !keychain.IsKeychainAvailable() {
		return nil, fmt.Errorf("macOS Keychain is not available")
	}

	if err := os.MkdirAll(storageDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	provider := &MacOSEncryptedProvider{
		StorageDir:      storageDir,
		ContextsFile:    filepath.Join(storageDir, "contexts.json"),
		keychainManager: keychain.NewManager(),
	}

	return provider, nil
}

// Load implements the Provider interface
func (p *MacOSEncryptedProvider) Load() ([]WithPath, error) {
	contexts, err := p.loadContextsMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to load contexts metadata: %w", err)
	}

	var kubeconfigs []WithPath
	for _, ctx := range contexts {
		kubeconfig, err := p.decryptKubeconfig(ctx)
		if err != nil {
			// Log error but continue with other contexts
			fmt.Fprintf(os.Stderr, "Warning: failed to decrypt context %s: %v\n", ctx.Name, err)
			continue
		}

		kubeconfigs = append(kubeconfigs, WithPath{
			Config:   kubeconfig,
			FilePath: ctx.EncryptedFile, // Use encrypted file path as identifier
		})
	}

	return kubeconfigs, nil
}

// EncryptKubeconfig encrypts a kubeconfig file and stores metadata
func (p *MacOSEncryptedProvider) EncryptKubeconfig(kubeconfigPath, contextName string) error {
	// Read the original kubeconfig
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	// Validate that the context exists in the kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	if _, exists := config.Contexts[contextName]; !exists {
		return fmt.Errorf("context %s not found in kubeconfig", contextName)
	}

	// Get or create encryption key for this context
	key, err := p.keychainManager.GetOrCreateKey(contextName)
	if err != nil {
		return fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Encrypt the kubeconfig data
	encryptedData, err := p.keychainManager.Encrypt(data, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt kubeconfig: %w", err)
	}

	// Generate encrypted file path
	encryptedFileName := fmt.Sprintf("%s.encrypted", sanitizeFilename(contextName))
	encryptedFilePath := filepath.Join(p.StorageDir, encryptedFileName)

	// Write encrypted data
	if err := os.WriteFile(encryptedFilePath, encryptedData, 0o600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	// Update contexts metadata
	if err := p.addContextMetadata(EncryptedContext{
		Name:          contextName,
		EncryptedFile: encryptedFilePath,
		OriginalFile:  kubeconfigPath,
	}); err != nil {
		// Clean up encrypted file if metadata update fails
		os.Remove(encryptedFilePath)
		return fmt.Errorf("failed to update context metadata: %w", err)
	}

	return nil
}

// RemoveEncryptedKubeconfig removes an encrypted kubeconfig and its metadata
func (p *MacOSEncryptedProvider) RemoveEncryptedKubeconfig(contextName string) error {
	contexts, err := p.loadContextsMetadata()
	if err != nil {
		return fmt.Errorf("failed to load contexts metadata: %w", err)
	}

	var foundContext *EncryptedContext
	var updatedContexts []EncryptedContext

	for _, ctx := range contexts {
		if ctx.Name == contextName {
			foundContext = &ctx
		} else {
			updatedContexts = append(updatedContexts, ctx)
		}
	}

	if foundContext == nil {
		return fmt.Errorf("encrypted context %s not found", contextName)
	}

	// Remove encrypted file
	if err := os.Remove(foundContext.EncryptedFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove encrypted file: %w", err)
	}

	// Remove encryption key from keychain
	if err := p.keychainManager.DeleteKey(contextName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove encryption key: %v\n", err)
	}

	// Update contexts metadata
	return p.saveContextsMetadata(updatedContexts)
}

// ListEncryptedContexts returns a list of encrypted context names
func (p *MacOSEncryptedProvider) ListEncryptedContexts() ([]string, error) {
	contexts, err := p.loadContextsMetadata()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, ctx := range contexts {
		names = append(names, ctx.Name)
	}

	return names, nil
}

// decryptKubeconfig decrypts a kubeconfig for the given context
func (p *MacOSEncryptedProvider) decryptKubeconfig(ctx EncryptedContext) (*api.Config, error) {
	// Read encrypted data
	encryptedData, err := os.ReadFile(ctx.EncryptedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Get decryption key
	key, err := p.keychainManager.GetOrCreateKey(ctx.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get decryption key: %w", err)
	}

	// Decrypt data
	data, err := p.keychainManager.Decrypt(encryptedData, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt kubeconfig: %w", err)
	}

	// Parse kubeconfig
	config, err := clientcmd.Load(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decrypted kubeconfig: %w", err)
	}

	return config, nil
}

// loadContextsMetadata loads the contexts metadata from disk
func (p *MacOSEncryptedProvider) loadContextsMetadata() ([]EncryptedContext, error) {
	if _, err := os.Stat(p.ContextsFile); os.IsNotExist(err) {
		return []EncryptedContext{}, nil
	}

	data, err := os.ReadFile(p.ContextsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read contexts file: %w", err)
	}

	var contexts []EncryptedContext
	if err := json.Unmarshal(data, &contexts); err != nil {
		return nil, fmt.Errorf("failed to parse contexts file: %w", err)
	}

	return contexts, nil
}

// saveContextsMetadata saves the contexts metadata to disk
func (p *MacOSEncryptedProvider) saveContextsMetadata(contexts []EncryptedContext) error {
	data, err := json.MarshalIndent(contexts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal contexts: %w", err)
	}

	if err := os.WriteFile(p.ContextsFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write contexts file: %w", err)
	}

	return nil
}

// addContextMetadata adds or updates context metadata
func (p *MacOSEncryptedProvider) addContextMetadata(newContext EncryptedContext) error {
	contexts, err := p.loadContextsMetadata()
	if err != nil {
		return err
	}

	// Check if context already exists and update it
	found := false
	for i, ctx := range contexts {
		if ctx.Name == newContext.Name {
			contexts[i] = newContext
			found = true
			break
		}
	}

	// If not found, add new context
	if !found {
		contexts = append(contexts, newContext)
	}

	return p.saveContextsMetadata(contexts)
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}
