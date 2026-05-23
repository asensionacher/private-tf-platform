package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

var gcm cipher.AEAD

// Init initializes the encryption module with a key
func Init() error {
	// Get encryption key from environment variable
	// In production, this should be a strong, randomly generated key stored securely
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		// Generate a default key if not provided (NOT RECOMMENDED FOR PRODUCTION)
		// In production, you should require this to be set
		key = "default-32-byte-encryption-key!!"
		fmt.Println("⚠️  WARNING: Using default encryption key. Set ENCRYPTION_KEY environment variable for production!")
	} else {
		fmt.Println("✓ Encryption initialized with custom key")
	}

	// Ensure key is exactly 32 bytes for AES-256
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		// Pad with zeros if too short
		paddedKey := make([]byte, 32)
		copy(paddedKey, keyBytes)
		keyBytes = paddedKey
	} else if len(keyBytes) > 32 {
		// Truncate if too long
		keyBytes = keyBytes[:32]
	}

	// Create AES cipher
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM (Galois/Counter Mode)
	gcm, err = cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	return nil
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext
func Encrypt(plaintext string) (string, error) {
	if gcm == nil {
		return "", fmt.Errorf("encryption not initialized")
	}

	if plaintext == "" {
		return "", nil
	}

	// Create a new nonce for each encryption
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext
func Decrypt(ciphertext string) (string, error) {
	if gcm == nil {
		return "", fmt.Errorf("encryption not initialized")
	}

	if ciphertext == "" {
		return "", nil
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Get nonce size
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, cipherBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptJSON encrypts a JSON string
func EncryptJSON(jsonStr string) (string, error) {
	return Encrypt(jsonStr)
}

// DecryptJSON decrypts a JSON string
func DecryptJSON(encryptedJSON string) (string, error) {
	return Decrypt(encryptedJSON)
}
