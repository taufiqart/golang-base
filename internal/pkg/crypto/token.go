package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

var ErrEmptyKey = errors.New("TOKEN_ENCRYPT_KEY not set")

// key returns 32-byte AES key from env, padded/truncated as needed.
func key() ([]byte, error) {
	k := os.Getenv("TOKEN_ENCRYPT_KEY")
	if k == "" {
		return nil, ErrEmptyKey
	}
	b := []byte(k)
	key := make([]byte, 32)
	copy(key, b) // truncate if >32, zero-pad if <32
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM, returns base64 ciphertext.
func Encrypt(plaintext string) (string, error) {
	k, err := key()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts base64 AES-256-GCM ciphertext, returns plaintext.
func Decrypt(ciphertext string) (string, error) {
	k, err := key()
	if err != nil {
		return "", err
	}
	data, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", errors.New("ciphertext too short")
	}
	plain, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
