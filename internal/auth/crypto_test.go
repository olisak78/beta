package auth

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustSetSecret(t *testing.T) string {
	t.Helper()
	// Deterministic 32-byte secret for tests
	secretBytes := bytes.Repeat([]byte{1}, 32)
	secret := base64.StdEncoding.EncodeToString(secretBytes)
	require.NoError(t, SetTokenSecret(secret))
	return secret
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	_ = mustSetSecret(t)

	plaintext := "my-token-123"
	ciphertext, err := EncryptToken(plaintext)
	require.NoError(t, err)
	require.NotEqual(t, plaintext, ciphertext)
	require.True(t, strings.HasPrefix(ciphertext, "enc:v1:"))

	decrypted, err := DecryptToken(ciphertext)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestDecryptPlaintextPassThrough(t *testing.T) {
	_ = mustSetSecret(t)

	plaintext := "plain-token"
	_, err := DecryptToken(plaintext)
	require.Error(t, err)
	require.Contains(t, err.Error(), "encrypted")
}

func TestSetTokenSecretInvalidLength(t *testing.T) {
	// 16 bytes instead of 32
	shortBytes := bytes.Repeat([]byte{2}, 16)
	short := base64.StdEncoding.EncodeToString(shortBytes)
	err := SetTokenSecret(short)
	require.Error(t, err)
}
