package auth

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type User struct {
	ID             int
	Username       string
	PasswordHash   string
	Name           string
	Role           string
	Permissions    []string
	OrganizationID int
	BranchID       *int
	CreatedAt      time.Time
}

type userRepository struct {
	db *sql.DB
}

type UserRepository interface {
	FindAll() ([]User, error)
	FindByID(id int) (*User, error)
	FindByUsername(username string) (*User, error)
	Create(u *User) error
	UpdatePassword(userID int, newHash string) error
	UpdateRole(userID int, role string, permissions []string) error
	CreateRefreshToken(userID int, tokenHash string, expiresAt time.Time) error
	FindUserByRefreshToken(tokenHash string) (*User, error)
	RevokeRefreshToken(tokenHash string) error
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindAll() ([]User, error) {
	rows, err := r.db.Query("SELECT id, username, password_hash, name, role, COALESCE(permissions, '{}'), COALESCE(organization_id, 0), branch_id, created_at FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, (*pq.StringArray)(&u.Permissions), &u.OrganizationID, &u.BranchID, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *userRepository) FindByID(id int) (*User, error) {
	row := r.db.QueryRow("SELECT id, username, password_hash, name, role, COALESCE(permissions, '{}'), COALESCE(organization_id, 0), branch_id, created_at FROM users WHERE id = $1", id)
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, (*pq.StringArray)(&u.Permissions), &u.OrganizationID, &u.BranchID, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) FindByUsername(username string) (*User, error) {
	row := r.db.QueryRow("SELECT id, username, password_hash, name, role, COALESCE(permissions, '{}'), COALESCE(organization_id, 0), branch_id, created_at FROM users WHERE username = $1", username)
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, (*pq.StringArray)(&u.Permissions), &u.OrganizationID, &u.BranchID, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) Create(u *User) error {
	return r.db.QueryRow(`INSERT INTO users (username, password_hash, name, role, organization_id, branch_id)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		u.Username, u.PasswordHash, u.Name, u.Role, u.OrganizationID, u.BranchID).Scan(&u.ID, &u.CreatedAt)
}

func (r *userRepository) UpdatePassword(userID int, newHash string) error {
	_, err := r.db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", newHash, userID)
	return err
}

func (r *userRepository) UpdateRole(userID int, role string, permissions []string) error {
	_, err := r.db.Exec("UPDATE users SET role = $1, permissions = $2 WHERE id = $3", role, pq.Array(permissions), userID)
	return err
}

func (r *userRepository) CreateRefreshToken(userID int, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec("INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)", userID, tokenHash, expiresAt)
	return err
}

func (r *userRepository) FindUserByRefreshToken(tokenHash string) (*User, error) {
	row := r.db.QueryRow(`SELECT u.id, u.username, u.password_hash, u.name, u.role,
			COALESCE(u.permissions, '{}'), COALESCE(u.organization_id, 0), u.branch_id, u.created_at
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = $1 AND rt.revoked_at IS NULL AND rt.expires_at > NOW()`, tokenHash)
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, (*pq.StringArray)(&u.Permissions), &u.OrganizationID, &u.BranchID, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) RevokeRefreshToken(tokenHash string) error {
	_, err := r.db.Exec("UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL", tokenHash)
	return err
}
