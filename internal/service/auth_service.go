package service

import (
	"context"
	"errors"
	"time"

	"go-kasir-api/internal/model"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByID(ctx context.Context, id int) (*model.User, error)
	CreateSession(ctx context.Context, userID int, duration time.Duration) (*model.Session, error)
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

const sessionDuration = 24 * time.Hour

type AuthService struct {
	repo UserRepository
}

func NewAuthService(repo UserRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*model.Session, *model.User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	session, err := s.repo.CreateSession(ctx, user.ID, sessionDuration)
	if err != nil {
		return nil, nil, err
	}

	return session, user, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) (*model.User, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, errors.New("invalid session")
	}

	user, err := s.repo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.repo.DeleteSession(ctx, sessionID)
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
