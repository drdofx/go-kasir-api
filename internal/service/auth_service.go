package service

import (
	"errors"
	"time"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

const sessionDuration = 24 * time.Hour

type AuthService struct {
	repo *repository.UserRepository
}

func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) Login(username, password string) (*model.Session, *model.User, error) {
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	session, err := s.repo.CreateSession(user.ID, sessionDuration)
	if err != nil {
		return nil, nil, err
	}

	return session, user, nil
}

func (s *AuthService) ValidateSession(sessionID string) (*model.User, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, errors.New("invalid session")
	}

	user, err := s.repo.GetByID(sess.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (s *AuthService) Logout(sessionID string) error {
	return s.repo.DeleteSession(sessionID)
}

// HashPassword creates a bcrypt hash for seeding users.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
