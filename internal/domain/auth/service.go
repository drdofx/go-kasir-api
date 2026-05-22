package auth

import (
    "errors"
    "fmt"
    "time"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

var (
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrInvalidToken       = errors.New("invalid token")
    ErrUserNotFound       = errors.New("user not found")
    ErrIncorrectPassword  = errors.New("current password is incorrect")
)

type AuthService struct {
    userRepo  UserRepository
    jwtSecret string
}

func NewAuthService(userRepo UserRepository, jwtSecret string) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (s *AuthService) Login(username, password string) (string, *User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		return "", nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", nil, err
	}
	return tokenStr, user, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (int, string, string, error) {
    token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return []byte(s.jwtSecret), nil
    })
    if err != nil || !token.Valid {
        return 0, "", "", ErrInvalidToken
    }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return 0, "", "", ErrInvalidToken
    }
    userIDFloat, ok := claims["user_id"].(float64)
    if !ok {
        return 0, "", "", ErrInvalidToken
    }
    username, ok := claims["username"].(string)
    if !ok {
        return 0, "", "", ErrInvalidToken
    }
    role, ok := claims["role"].(string)
    if !ok {
        return 0, "", "", ErrInvalidToken
    }
    return int(userIDFloat), username, role, nil
}

func (s *AuthService) ChangePassword(userID int, currentPassword, newPassword string) error {
    user, err := s.userRepo.FindByID(userID)
    if err != nil {
        return fmt.Errorf("find user: %w", err)
    }
    if user == nil {
        return ErrUserNotFound
    }
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
        return ErrIncorrectPassword
    }
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(userID, string(hash))
}
