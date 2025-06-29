package keychain

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

const (
	keychainService = "kubert-encrypted-kubeconfigs"
	keySize         = 32 // AES-256
)

// Manager handles encryption/decryption using macOS Keychain
type Manager struct {
	serviceName string
}

// NewManager creates a new keychain manager
func NewManager() *Manager {
	return &Manager{
		serviceName: keychainService,
	}
}

// GetOrCreateKey retrieves or creates an encryption key for the given context
func (m *Manager) GetOrCreateKey(contextName string) ([]byte, error) {
	// Try to get existing key first
	key, err := m.getKey(contextName)
	if err == nil {
		return key, nil
	}

	// If key doesn't exist, create a new one
	key = make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Store the key in keychain
	if err := m.storeKey(contextName, key); err != nil {
		return nil, fmt.Errorf("failed to store encryption key: %w", err)
	}

	return key, nil
}

// DeleteKey removes the encryption key for the given context
func (m *Manager) DeleteKey(contextName string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", m.serviceName,
		"-a", contextName)

	if err := cmd.Run(); err != nil {
		// Ignore "item not found" errors
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 44 {
			return nil
		}
		return fmt.Errorf("failed to delete key from keychain: %w", err)
	}

	return nil
}

// getKey retrieves an encryption key from macOS Keychain
func (m *Manager) getKey(contextName string) ([]byte, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", m.serviceName,
		"-a", contextName,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key from keychain: %w", err)
	}

	keyBase64 := strings.TrimSpace(string(output))
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key from keychain: %w", err)
	}

	return key, nil
}

// storeKey stores an encryption key in macOS Keychain
func (m *Manager) storeKey(contextName string, key []byte) error {
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	cmd := exec.Command("security", "add-generic-password",
		"-s", m.serviceName,
		"-a", contextName,
		"-w", keyBase64,
		"-U") // Update if exists

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store key in keychain: %w", err)
	}

	return nil
}

// Encrypt encrypts data using AES-GCM
func (m *Manager) Encrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (m *Manager) Decrypt(encryptedData []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(encryptedData) < gcm.NonceSize() {
		return nil, fmt.Errorf("encrypted data too short")
	}

	nonce := encryptedData[:gcm.NonceSize()]
	ciphertext := encryptedData[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// IsKeychainAvailable checks if macOS Keychain is available
func IsKeychainAvailable() bool {
	// Check if we're on macOS and security command is available
	if _, err := exec.LookPath("security"); err != nil {
		return false
	}

	// Check if we can access the keychain
	cmd := exec.Command("security", "list-keychains")
	return cmd.Run() == nil
}
