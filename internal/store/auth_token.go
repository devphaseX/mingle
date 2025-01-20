package store

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/o1egl/paseto"
)

type TokenMaker interface {
	GenerateAccessToken(userID, sessionID string) (string, error)
	GenerateRefreshToken(sessionID string) (string, error)
	ValidateAccessToken(tokenString string) (*AccessPayload, error)
	ValidateRefreshToken(tokenString string) (*RefreshPayload, error)
}

type TokenStore struct {
	paseto        *paseto.V2
	accessKey     []byte // Symmetric key for access tokens
	refreshKey    []byte // Symmetric key for refresh tokens
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewTokenStore(accessSecret, refreshSecret string, accessExpiry, refreshExpiry, rememberMeExpiry time.Duration) (*TokenStore, error) {

	// Decode the base64-encoded key
	accessSecretByte, err := base64.StdEncoding.DecodeString(accessSecret)
	if err != nil {
		fmt.Println("Failed to decode access secret base64 key:", err)
		return nil, err
	}

	// Decode the base64-encoded key
	refreshSecretByte, err := base64.StdEncoding.DecodeString(refreshSecret)
	if err != nil {
		fmt.Println("Failed to decode refresh secret base64 key:", err)
		return nil, err
	}

	// Verify access key length
	if len(accessSecretByte) != chacha20poly1305.KeySize {
		return nil, errors.New(fmt.Sprintf("invalid access key size: must be exactly %d bytes", chacha20poly1305.KeySize))
	}

	// Verify refresh key length
	if len(refreshSecretByte) != chacha20poly1305.KeySize {
		return nil, errors.New(fmt.Sprintf("invalid refresh key size: must be exactly %d bytes", chacha20poly1305.KeySize))
	}

	return &TokenStore{
		paseto:        paseto.NewV2(),
		accessKey:     accessSecretByte,
		refreshKey:    refreshSecretByte,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}, nil
}

// Payload for access tokens
type AccessPayload struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

// Payload for refresh tokens
type RefreshPayload struct {
	SessionID string `json:"session_id"`
}

// GenerateAccessToken creates a PASETO token for access
func (t *TokenStore) GenerateAccessToken(userID, sessionID string) (string, error) {
	payload := AccessPayload{
		UserID:    userID,
		SessionID: sessionID,
	}

	// Set token expiration
	expiration := time.Now().Add(t.accessExpiry)

	// Create the token
	token, err := t.paseto.Encrypt(t.accessKey, payload, expiration)
	if err != nil {
		return "", err
	}

	return token, nil
}

// GenerateRefreshToken creates a PASETO token for refresh
func (t *TokenStore) GenerateRefreshToken(sessionID string) (string, error) {
	payload := RefreshPayload{
		SessionID: sessionID,
	}

	// Set token expiration
	expiration := time.Now().Add(t.refreshExpiry)

	// Create the token
	token, err := t.paseto.Encrypt(t.refreshKey, payload, expiration)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidateAccessToken validates a PASETO access token
func (t *TokenStore) ValidateAccessToken(tokenString string) (*AccessPayload, error) {
	var payload AccessPayload

	// Decrypt and validate the token
	err := t.paseto.Decrypt(tokenString, t.accessKey, &payload, nil)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}

// ValidateRefreshToken validates a PASETO refresh token
func (t *TokenStore) ValidateRefreshToken(tokenString string) (*RefreshPayload, error) {
	var payload RefreshPayload

	// Decrypt and validate the token
	err := t.paseto.Decrypt(tokenString, t.refreshKey, &payload, nil)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}
