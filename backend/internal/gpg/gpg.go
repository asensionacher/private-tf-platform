package gpg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	gpgHome     string
	keyID       string
	publicKey   string
	initialized bool
	initMutex   sync.Mutex
)

const (
	keyName  = "Terraform Registry"
	keyEmail = "registry@localhost"
)

// Init initializes GPG with a signing key
func Init() error {
	initMutex.Lock()
	defer initMutex.Unlock()

	if initialized {
		return nil
	}

	// Set GPG home directory
	gpgHome = os.Getenv("GPG_HOME")
	if gpgHome == "" {
		gpgHome = "/app/data/gpg"
	}

	// Create GPG home directory
	if err := os.MkdirAll(gpgHome, 0700); err != nil {
		return fmt.Errorf("failed to create GPG home: %w", err)
	}

	// Check if key already exists
	existingKeyID, err := getExistingKeyID()
	if err == nil && existingKeyID != "" {
		keyID = existingKeyID
		publicKey, _ = exportPublicKey(keyID)
		initialized = true
		return nil
	}

	// Generate new key
	if err := generateKey(); err != nil {
		return fmt.Errorf("failed to generate GPG key: %w", err)
	}

	// Get the key ID
	keyID, err = getExistingKeyID()
	if err != nil {
		return fmt.Errorf("failed to get key ID: %w", err)
	}

	// Export public key
	publicKey, err = exportPublicKey(keyID)
	if err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
	}

	initialized = true
	return nil
}

func getExistingKeyID() (string, error) {
	cmd := exec.Command("gpg", "--homedir", gpgHome, "--list-keys", "--keyid-format", "long", keyEmail)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output to find key ID
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "pub") {
			// Format: pub   rsa4096/KEYID 2024-01-01 [SC]
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				keyPart := parts[1]
				if idx := strings.Index(keyPart, "/"); idx != -1 {
					return keyPart[idx+1:], nil
				}
			}
		}
	}

	return "", fmt.Errorf("key not found")
}

func generateKey() error {
	// Create batch file for unattended key generation
	batchContent := fmt.Sprintf(`%%no-protection
Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: %s
Name-Email: %s
Expire-Date: 0
%%commit
`, keyName, keyEmail)

	batchFile := filepath.Join(gpgHome, "keygen.batch")
	if err := os.WriteFile(batchFile, []byte(batchContent), 0600); err != nil {
		return err
	}
	defer os.Remove(batchFile)

	cmd := exec.Command("gpg", "--homedir", gpgHome, "--batch", "--gen-key", batchFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpg keygen failed: %s", stderr.String())
	}

	return nil
}

func exportPublicKey(keyID string) (string, error) {
	cmd := exec.Command("gpg", "--homedir", gpgHome, "--armor", "--export", keyID)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// Sign signs the given data and returns the detached binary signature
func Sign(data string) ([]byte, error) {
	if !initialized {
		if err := Init(); err != nil {
			return nil, err
		}
	}

	cmd := exec.Command("gpg", "--homedir", gpgHome, "--detach-sign", "-u", keyID)
	cmd.Stdin = strings.NewReader(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg sign failed: %s", stderr.String())
	}

	return stdout.Bytes(), nil
}

// GetKeyID returns the GPG key ID
func GetKeyID() string {
	return keyID
}

// GetPublicKey returns the ASCII-armored public key
func GetPublicKey() string {
	return publicKey
}

// GetSigningKeys returns the signing keys structure for Terraform
func GetSigningKeys() map[string]interface{} {
	if !initialized || keyID == "" {
		return nil
	}

	return map[string]interface{}{
		"gpg_public_keys": []map[string]string{
			{
				"key_id":      keyID,
				"ascii_armor": publicKey,
			},
		},
	}
}
