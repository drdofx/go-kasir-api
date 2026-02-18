package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"go-kasir-api/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, name, role, created_at FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) GetByID(id int) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, name, role, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ---------- Sessions ----------

func (r *UserRepository) CreateSession(userID int, duration time.Duration) (*model.Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	s := &model.Session{
		ID:        hex.EncodeToString(b),
		UserID:    userID,
		ExpiresAt: time.Now().Add(duration),
	}
	_, err := r.db.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		s.ID, s.UserID, s.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *UserRepository) GetSession(sessionID string) (*model.Session, error) {
	s := &model.Session{}
	err := r.db.QueryRow(
		`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = $1 AND expires_at > NOW()`,
		sessionID,
	).Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *UserRepository) DeleteSession(sessionID string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

// CleanExpiredSessions removes expired sessions.
func (r *UserRepository) CleanExpiredSessions() error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at <= NOW()`)
	return err
}
