package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
	ErrBranchNotFound     = errors.New("branch not found")
	ErrBranchNotAllowed   = errors.New("branch does not belong to user's organization")
)

type AuthService struct {
	userRepo        UserRepository
	branchRepo      BranchRepository
	jwtSecret       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type BranchRepository interface {
	FindByID(id int) (*Branch, error)
}

type Branch struct {
	ID             int
	OrganizationID int
}

func NewAuthService(userRepo UserRepository, branchRepo BranchRepository, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		branchRepo:      branchRepo,
		jwtSecret:       jwtSecret,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Login(username, password string) (TokenPair, *User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return TokenPair{}, nil, err
	}
	if user == nil {
		return TokenPair{}, nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, nil, errors.New("invalid credentials")
	}
	accessToken, err := s.signAccessToken(user)
	if err != nil {
		return TokenPair{}, nil, err
	}
	refreshToken, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, nil, err
	}
	if err := s.userRepo.CreateRefreshToken(user.ID, hashToken(refreshToken), time.Now().Add(s.refreshTokenTTL)); err != nil {
		return TokenPair{}, nil, err
	}
	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, user, nil
}

func (s *AuthService) signAccessToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":     user.ID,
		"username":    user.Username,
		"role":        user.Role,
		"permissions": user.Permissions,
		"org_id":      user.OrganizationID,
		"exp":         time.Now().Add(s.accessTokenTTL).Unix(),
	}
	if user.BranchID != nil {
		claims["branch_id"] = *user.BranchID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateToken(tokenStr string) (int, string, string, []string, int, *int, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, "", "", nil, 0, nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", "", nil, 0, nil, ErrInvalidToken
	}
	uid, ok := claims["user_id"].(float64)
	if !ok {
		return 0, "", "", nil, 0, nil, ErrInvalidToken
	}
	username, ok := claims["username"].(string)
	if !ok {
		return 0, "", "", nil, 0, nil, ErrInvalidToken
	}
	role, ok := claims["role"].(string)
	if !ok {
		return 0, "", "", nil, 0, nil, ErrInvalidToken
	}
	orgID, ok := claims["org_id"].(float64)
	if !ok {
		return 0, "", "", nil, 0, nil, ErrInvalidToken
	}
	permissions := parsePermissionsClaim(claims["permissions"])
	var branchID *int
	if bid, ok := claims["branch_id"].(float64); ok {
		b := int(bid)
		branchID = &b
	}
	return int(uid), username, role, permissions, int(orgID), branchID, nil
}

func (s *AuthService) FindAllUsers() ([]User, error) {
	return s.userRepo.FindAll()
}

func (s *AuthService) SwitchBranch(userID, newBranchID int) (string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	branch, err := s.branchRepo.FindByID(newBranchID)
	if err != nil {
		return "", err
	}
	if branch == nil {
		return "", ErrBranchNotFound
	}
	if branch.OrganizationID != user.OrganizationID {
		return "", ErrBranchNotAllowed
	}
	claims := jwt.MapClaims{
		"user_id":     user.ID,
		"username":    user.Username,
		"role":        user.Role,
		"permissions": user.Permissions,
		"org_id":      user.OrganizationID,
		"branch_id":   newBranchID,
		"exp":         time.Now().Add(s.accessTokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) Refresh(refreshToken string) (string, error) {
	user, err := s.userRepo.FindUserByRefreshToken(hashToken(refreshToken))
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrInvalidToken
	}
	return s.signAccessToken(user)
}

func (s *AuthService) Logout(refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.userRepo.RevokeRefreshToken(hashToken(refreshToken))
}

func (s *AuthService) CreateUser(username, password, name, role string, organizationID int, branchID *int) (*User, error) {
	existing, _ := s.userRepo.FindByUsername(username)
	if existing != nil {
		return nil, errors.New("username already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &User{Username: username, PasswordHash: string(hash), Name: name, Role: role, OrganizationID: organizationID, BranchID: branchID}
	if u.Role == "" {
		u.Role = "cashier"
	}
	err = s.userRepo.Create(u)
	return u, err
}

func (s *AuthService) UpdateUserRole(userID int, role string, permissions []string) error {
	return s.userRepo.UpdateRole(userID, role, permissions)
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

func parsePermissionsClaim(value interface{}) []string {
	values, ok := value.([]interface{})
	if !ok {
		return []string{}
	}
	permissions := make([]string, 0, len(values))
	for _, value := range values {
		permission, ok := value.(string)
		if ok && permission != "" {
			permissions = append(permissions, permission)
		}
	}
	return permissions
}

func newRefreshToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
