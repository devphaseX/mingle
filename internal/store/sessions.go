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
	ID        string     `db:"id"`
	UserID    string     `db:"user_id"`
	UserAgent string     `db:"user_agent"`
	IP        string     `db:"ip"`
	ExpiresAt time.Time  `db:"expires_at"`
	LastUsed  *time.Time `db:"last_used"`
	CreatedAt time.Time  `db:"created_at"`
}

type SessionStore struct {
	db *sql.DB
}

func (s *SessionStore) CreateSession(ctx context.Context, userID, userAgent, ip string) (*Session, error) {
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		UserAgent: userAgent,
		IP:        ip,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 7), // 1 week
	}

	query := `INSERT INTO sessions (id, user_id, user_agent, ip, expires_at)
	          VALUES ($1, $2, $3 , $4, $5)`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := s.db.ExecContext(ctx, query, session.ID, session.UserID, session.UserAgent, session.IP, session.ExpiresAt)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *SessionStore) ValidateSession(ctx context.Context, sessionID string) (*Session, *User, error) {
	var session Session
	var user User
	var emailVerifiedAt sql.NullTime // Use sql.NullTime for nullable fields

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
		&session.ID, &session.UserID, &session.UserAgent, &session.IP, &session.ExpiresAt, &session.LastUsed, &session.CreatedAt,
		&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.IsActive, &emailVerifiedAt, &user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	// Handle nullable fields
	if emailVerifiedAt.Valid {
		user.EmailVerifiedAt = &emailVerifiedAt.Time
	} else {
		user.EmailVerifiedAt = nil
	}

	// Check if the session is expired
	if time.Now().After(session.ExpiresAt) {
		_ = s.InvalidateSession(ctx, sessionID)
		return nil, nil, nil
	}

	return &session, &user, nil
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
		&session.ID, &session.UserID, &session.UserAgent, &session.IP, &session.ExpiresAt, &session.LastUsed, &session.CreatedAt,
		&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.IsActive, &emailVerifiedAt, &user.CreatedAt,
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
