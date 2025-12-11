package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"sync"
)

const (
	// encPrefix helps us detect if a token is encrypted and supports future versioning
	encPrefix   = "enc:v1:"
	gcmNonceLen = 12 // AES-GCM standard nonce length
)

var (
	keyOnce       sync.Once
	keyInitErr    error
	encryptionKey []byte
)

// initKeyFromEnv lazily reads TOKEN_SECRET from the environment and prepares the AES-256 key.
// TOKEN_SECRET is expected to be a base64 encoded 32-byte value (openssl rand -base64 32).
func initKeyFromEnv() error {
	keyOnce.Do(func() {
		secret := os.Getenv("TOKEN_SECRET")
		if strings.TrimSpace(secret) == "" {
			keyInitErr = errors.New("TOKEN_SECRET env var is empty")
			return
		}
		raw, err := base64.StdEncoding.DecodeString(secret)
		if err != nil {
			keyInitErr = errors.New("failed to base64 decode TOKEN_SECRET")
			return
		}
		if len(raw) != 32 {
			keyInitErr = errors.New("TOKEN_SECRET must decode to exactly 32 bytes")
			return
		}
		encryptionKey = raw
	})
	return keyInitErr
}

// SetTokenSecret allows tests (or controlled initialization) to set the key explicitly.
// The provided secret must be a base64-encoded 32-byte string.
func SetTokenSecret(base64Secret string) error {
	raw, err := base64.StdEncoding.DecodeString(base64Secret)
	if err != nil {
		return errors.New("failed to base64 decode provided secret")
	}
	if len(raw) != 32 {
		return errors.New("provided secret must decode to exactly 32 bytes")
	}
	// Reset lazy init state
	keyOnce = sync.Once{}
	encryptionKey = raw
	keyInitErr = nil
	return nil
}

// EncryptToken encrypts the plaintext token using AES-256-GCM and returns an encoded string.
// Format: "enc:v1:" + base64(nonce || ciphertext)
func EncryptToken(plaintext string) (string, error) {
	if err := ensureKey(); err != nil {
		return "", err
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcmNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	combined := append(nonce, ciphertext...)
	return encPrefix + base64.StdEncoding.EncodeToString(combined), nil
}

// DecryptToken decrypts an encrypted token. Returns an error if the input does not have the expected prefix.
func DecryptToken(s string) (string, error) {
	// Strict mode: require encryption prefix, no plaintext backward compatibility (constant-time)
	if len(s) < len(encPrefix) || subtle.ConstantTimeCompare([]byte(s[:len(encPrefix)]), []byte(encPrefix)) != 1 {
		return "", errors.New("token is not encrypted (missing " + encPrefix + " prefix)")
	}

	if err := ensureKey(); err != nil {
		return "", err
	}

	encoded := s[len(encPrefix):]
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("failed to base64 decode encrypted token")
	}
	if len(combined) < gcmNonceLen {
		return "", errors.New("invalid encrypted token payload")
	}
	nonce := combined[:gcmNonceLen]
	ciphertext := combined[gcmNonceLen:]

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("failed to decrypt token")
	}
	return string(plain), nil
}

func ensureKey() error {
	// If already set via SetTokenSecret, skip env init
	if len(encryptionKey) == 32 && keyInitErr == nil {
		return nil
	}
	return initKeyFromEnv()
}
