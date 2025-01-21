package store

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/golang-jwt/jwt/v5"
	"github.com/o1egl/paseto"
)

var (
	ErrExpiredToken          = errors.New("token has expired")
	ErrInvalidToken          = errors.New("token not valid")
	ErrInvalidOrExpiredToken = errors.New("token expired or invalid")
	ErrUnverifiableToken     = errors.New("token is unverifiable")
)

type TokenMaker interface {
	GenerateAccessToken(userID int64, sessionID string, expiry time.Duration) (string, error)
	GenerateRefreshToken(sessionID string, version int, expiry time.Duration) (string, error)
	ValidateAccessToken(tokenString string) (*AccessPayload, error)
	ValidateRefreshToken(tokenString string) (*RefreshPayload, error)
}

type TokenStore struct {
	paseto     *paseto.V2
	accessKey  []byte // Symmetric key for access tokens
	refreshKey []byte // Symmetric key for refresh tokens

}

func NewTokenStore(accessSecret, refreshSecret string) (*TokenStore, error) {

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
		paseto:     paseto.NewV2(),
		accessKey:  accessSecretByte,
		refreshKey: refreshSecretByte,
	}, nil
}

// Payload for access tokens
type AccessPayload struct {
	UserID    int64  `json:"user_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func NewAccessPayload(userId int64, sessionId string, expiry time.Duration) *AccessPayload {
	return &AccessPayload{
		UserID:    userId,
		SessionID: sessionId,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
	}
}

func (p *AccessPayload) Valid() error {
	if time.Now().After(p.ExpiresAt.Time) {
		return ErrExpiredToken
	}

	return nil
}

// Payload for refresh tokens
type RefreshPayload struct {
	SessionID string `json:"session_id"`
	Version   int    `json:"version"`
	jwt.RegisteredClaims
}

func NewRefreshPayload(sessionId string, version int, expiry time.Duration) *RefreshPayload {
	return &RefreshPayload{
		SessionID: sessionId,
		Version:   version,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
	}
}

func (p *RefreshPayload) Valid() error {
	if time.Now().After(p.ExpiresAt.Time) {
		return ErrExpiredToken
	}

	return nil
}

// GenerateAccessToken creates a PASETO token for access
func (t *TokenStore) GenerateAccessToken(userID int64, sessionID string, accessExpiry time.Duration) (string, error) {
	payload := NewAccessPayload(userID, sessionID, accessExpiry)

	// Set token expiration

	// Create the token
	token, err := t.paseto.Encrypt(t.accessKey, payload, nil)
	if err != nil {
		return "", err
	}

	return token, nil
}

// GenerateRefreshToken creates a PASETO token for refresh
func (t *TokenStore) GenerateRefreshToken(sessionID string, version int, refreshExpiry time.Duration) (string, error) {
	payload := NewRefreshPayload(sessionID, version, refreshExpiry)
	// Set token expiration

	// Create the token
	token, err := t.paseto.Encrypt(t.refreshKey, payload, nil)
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
		return nil, ErrInvalidToken
	}

	return &payload, nil
}

// ValidateRefreshToken validates a PASETO refresh token
func (t *TokenStore) ValidateRefreshToken(tokenString string) (*RefreshPayload, error) {
	var payload RefreshPayload

	// Decrypt and validate the token
	err := t.paseto.Decrypt(tokenString, t.refreshKey, &payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &payload, nil
}
