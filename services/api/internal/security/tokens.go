package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

type TokenProtector interface {
	Protect(ctx context.Context, plaintext string) (string, error)
	Unprotect(ctx context.Context, ciphertext string) (string, error)
}

type AESGCMTokenProtector struct {
	key []byte
}

func NewAESGCMTokenProtector(rawKey string) (*AESGCMTokenProtector, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return nil, errors.New("provider token encryption key is required")
	}
	if decoded, err := base64.StdEncoding.DecodeString(rawKey); err == nil && len(decoded) >= 32 {
		return &AESGCMTokenProtector{key: decoded[:32]}, nil
	}
	sum := sha256.Sum256([]byte(rawKey))
	return &AESGCMTokenProtector{key: sum[:]}, nil
}

func (p *AESGCMTokenProtector) Protect(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(p.key)
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
	return "v1:" + base64.StdEncoding.EncodeToString(sealed), nil
}

func (p *AESGCMTokenProtector) Unprotect(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !strings.HasPrefix(ciphertext, "v1:") {
		return "", errors.New("unsupported token ciphertext")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "v1:"))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("invalid token ciphertext")
	}
	nonce, payload := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

type DevelopmentTokenProtector struct{}

func (DevelopmentTokenProtector) Protect(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	return "dev:" + base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (DevelopmentTokenProtector) Unprotect(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !strings.HasPrefix(ciphertext, "dev:") {
		return "", errors.New("unsupported development token ciphertext")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "dev:"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func NewTokenProtector(appEnv, rawKey string) (TokenProtector, error) {
	if strings.TrimSpace(rawKey) != "" {
		return NewAESGCMTokenProtector(rawKey)
	}
	if appEnv == "production" {
		return nil, errors.New("PROVIDER_TOKEN_ENCRYPTION_KEY is required in production")
	}
	return DevelopmentTokenProtector{}, nil
}
