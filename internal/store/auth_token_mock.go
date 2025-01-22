package store

import "time"

type TestTokenStore struct {
}

func (t *TestTokenStore) GenerateAccessToken(userID int64, sessionID string, accessExpiry time.Duration) (string, error) {
	return "", nil
}

func (t *TestTokenStore) GenerateRefreshToken(sessionID string, version int, refreshExpiry time.Duration) (string, error) {
	return "", nil
}

func (t *TestTokenStore) ValidateAccessToken(tokenString string) (*AccessPayload, error) {
	return &AccessPayload{}, nil
}

func (t *TestTokenStore) ValidateRefreshToken(tokenString string) (*RefreshPayload, error) {
	return &RefreshPayload{}, nil
}
