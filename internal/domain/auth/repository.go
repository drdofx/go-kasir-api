package auth

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
}

type userRepository struct {
	db *sql.DB
}

type UserRepository interface {
	FindByID(id int) (*User, error)
	FindByUsername(username string) (*User, error)
	UpdatePassword(userID int, newHash string) error
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByID(id int) (*User, error) {
	row := r.db.QueryRow("SELECT id, username, password_hash, name, role, created_at FROM users WHERE id = $1", id)
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) FindByUsername(username string) (*User, error) {
	row := r.db.QueryRow("SELECT id, username, password_hash, name, role, created_at FROM users WHERE username = $1", username)
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) UpdatePassword(userID int, newHash string) error {
	_, err := r.db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", newHash, userID)
	return err
}
