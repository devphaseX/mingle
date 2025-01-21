package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID                 string     `json:"id"`
	UserID             int64      `json:"user_id"`
	UserAgent          string     `json:"user_agent"`
	IP                 string     `json:"ip"`
	Version            int        `json:"version"`
	ExpiresAt          time.Time  `json:"expires_at"`
	LastUsed           *time.Time `json:"last_used"`
	CreatedAt          time.Time  `json:"created_at"`
	RememberMe         bool       `json:"remember_me"`          // Whether the session should be extended
	MaxRenewalDuration int64      `json:"max_renewal_duration"` // Maximum duration for session renewal (in seconds)
}

type SessionStore struct {
	db *sql.DB
}

func (s *SessionStore) CreateSession(ctx context.Context, userID int64, userAgent, ip string, expiry time.Duration, rememberMe bool) (*Session, error) {
	session := &Session{
		ID:         uuid.New().String(),
		UserID:     userID,
		UserAgent:  userAgent,
		IP:         ip,
		Version:    1,
		RememberMe: rememberMe,
		ExpiresAt:  time.Now().Add(expiry), // 1 week
	}

	query := `INSERT INTO sessions (id, user_id, user_agent, ip, expires_at)
	          VALUES ($1, $2, $3 , $4, $5) RETURNING version`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := s.db.QueryRowContext(ctx,
		query,
		session.ID,
		session.UserID,
		session.UserAgent,
		session.IP,
		session.ExpiresAt,
	).Scan(&session.Version)

	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *SessionStore) ValidateSession(ctx context.Context, sessionID string, version int) (*Session, *User, bool, error) {
	var session Session
	var user User
	var emailVerifiedAt sql.NullTime
	var maxRenewalDuration sql.NullInt64

	query := `
		SELECT
			s.id, s.user_id, s.user_agent, s.ip, s.expires_at, s.last_used, s.created_at, s.remember_me, s.max_renewal_duration,
			u.id, u.first_name, u.last_name, u.username, u.email, u.is_active, u.email_verified_at, u.created_at
		FROM sessions s
		INNER JOIN users u ON s.user_id = u.id
		WHERE s.id = $1 and s.version = $2
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	row := s.db.QueryRowContext(ctx, query, sessionID, version)

	err := row.Scan(
		&session.ID, &session.UserID, &session.UserAgent, &session.IP, &session.ExpiresAt, &session.LastUsed, &session.CreatedAt, &session.RememberMe, &maxRenewalDuration,
		&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.IsActive, &emailVerifiedAt, &user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, false, nil
		}
		return nil, nil, false, err
	}

	// Handle nullable fields
	if emailVerifiedAt.Valid {
		user.EmailVerifiedAt = &emailVerifiedAt.Time
	} else {
		user.EmailVerifiedAt = nil
	}

	if maxRenewalDuration.Valid {
		session.MaxRenewalDuration = maxRenewalDuration.Int64
	}
	// Check if the session is expired
	now := time.Now()
	if now.After(session.ExpiresAt) {
		_ = s.InvalidateSession(ctx, sessionID)
		return nil, nil, false, nil
	}

	// Check if the session can be extended (Remember Me is enabled)
	canExtend := false
	if session.RememberMe {
		// Calculate the maximum allowed expiration time
		maxRenewalTime := session.CreatedAt.Add(time.Duration(session.MaxRenewalDuration) * time.Second)

		// If the current expiration time is before the maximum allowed, the session can be extended
		if session.ExpiresAt.Before(maxRenewalTime) {
			canExtend = true
		} else {
			// The session has exceeded the maximum renewal duration; force the user to log in again
			_ = s.InvalidateSession(ctx, sessionID)
			return nil, nil, false, nil
		}
	}

	return &session, &user, canExtend, nil
}

func (s *SessionStore) InvalidateSession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM sessions WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := s.db.Exec(query, sessionID)
	return err
}

func (s *SessionStore) UpdateLastUsed(ctx context.Context, sessionID string) error {
	query := `UPDATE sessions SET last_used = NOW() WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := s.db.Exec(query, sessionID)
	return err
}

func (s *SessionStore) GetSessionByID(ctx context.Context, sessionID string) (*Session, *User, error) {
	var session Session
	var user User
	var emailVerifiedAt sql.NullTime

	query := `
		SELECT
			s.id, s.user_id, s.user_agent, s.ip, s.expires_at, s.last_used, s.created_at,
			u.id, u.first_name, u.last_name, u.username, u.email, u.is_active, u.email_verified_at, u.created_at
		FROM sessions s
		INNER JOIN users u ON s.user_id = u.id
		WHERE s.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	row := s.db.QueryRowContext(ctx, query, sessionID)

	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.UserAgent,
		&session.IP,
		&session.ExpiresAt,
		&session.LastUsed,
		&session.CreatedAt,
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Email,
		&user.IsActive,
		&emailVerifiedAt,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("session not found")
		}
		return nil, nil, err
	}

	// Handle nullable fields
	if emailVerifiedAt.Valid {
		user.EmailVerifiedAt = &emailVerifiedAt.Time
	} else {
		user.EmailVerifiedAt = nil
	}

	return &session, &user, nil
}

func (s *SessionStore) GetSessionsByUserID(ctx context.Context, userID string, isAdmin bool, paginateQuery PaginateQueryFilter) ([]Session, Metadata, error) {
	var sessions []Session
	var totalRecords int

	query := `
		SELECT count(*) OVER(), id, user_id, user_agent, ip, expires_at, last_used, created_at
		FROM sessions
	`
	if !isAdmin {
		query += ` WHERE user_id = $1`
	}

	query += fmt.Sprintf(` ORDER BY %s %s LIMIT $2 OFFSET $3`, paginateQuery.SortColumn(), paginateQuery.SortDirection())

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var rows *sql.Rows
	var err error
	if !isAdmin {
		rows, err = s.db.QueryContext(ctx, query, userID, paginateQuery.Limit(), paginateQuery.Offset())
	} else {
		rows, err = s.db.QueryContext(ctx, query, paginateQuery.Limit(), paginateQuery.Offset())
	}
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var session Session
		err := rows.Scan(
			&totalRecords,
			&session.ID, &session.UserID, &session.UserAgent, &session.IP, &session.ExpiresAt, &session.LastUsed, &session.CreatedAt,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, paginateQuery.Page, paginateQuery.PageSize)

	return sessions, metadata, nil
}

func (s *SessionStore) ExtendSessionAndGenerateRefreshToken(ctx context.Context, session *Session, tokenMaker TokenMaker, rememberPeriod time.Duration) (string, error) {
	// Check if RememberMe is enabled
	if !session.RememberMe {
		return "", ErrSessionCannotBeExtends
	}

	// Calculate the maximum allowed expiration time
	maxRenewalTime := session.CreatedAt.Add(time.Duration(session.MaxRenewalDuration) * time.Second)

	// Calculate the new expiration time (current time + remember period)
	newExpiresAt := time.Now().Add(rememberPeriod)

	// Ensure the new expiration time does not exceed the maximum renewal duration
	if newExpiresAt.After(maxRenewalTime) {
		newExpiresAt = maxRenewalTime
	}

	var version int
	// Update the session expiration time and refresh token hash in the database
	updateQuery := `UPDATE sessions SET expires_at = $1, version = version + 1 WHERE id = $2 AND version = $3 RETURNING version`
	err := s.db.QueryRowContext(ctx, updateQuery, newExpiresAt, session.ID, session.Version).Scan(&version)
	if err != nil {
		return "", fmt.Errorf("failed to extend session: %w", err)
	}

	// Generate a new refresh token
	newRefreshToken, err := tokenMaker.GenerateRefreshToken(session.ID, version, rememberPeriod)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update the session's ExpiresAt field in memory
	session.ExpiresAt = newExpiresAt

	// Return the new refresh token (unhashed) to the client
	return newRefreshToken, nil
}
