package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务。
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret []byte
	expire    time.Duration
	logger    *slog.Logger
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, expireHours int, logger *slog.Logger) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		expire:    time.Duration(expireHours) * time.Hour,
		logger:    logger,
	}
}

// Claims JWT 载荷。
type Claims struct {
	UserID uint   `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Login 登录。
func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", fmt.Errorf("invalid username or password")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", fmt.Errorf("invalid username or password")
	}
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", err
	}
	s.logger.Info("user logged in", "username", user.Username)
	return token, nil
}

// GenerateToken 签发 JWT。
func (s *AuthService) GenerateToken(user *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expire)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return token, nil
}

// jwtSignWithClaims 生成指定过期时间的测试用 token。
func (s *AuthService) jwtSignWithClaims(expiresAt time.Time) (string, error) {
	claims := Claims{UserID: 1, Role: "admin", RegisteredClaims: jwt.RegisteredClaims{Subject: "admin", ExpiresAt: jwt.NewNumericDate(expiresAt)}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// ParseToken 解析 JWT。
func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return claims, nil
}
