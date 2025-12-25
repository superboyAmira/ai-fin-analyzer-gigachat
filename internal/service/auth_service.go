package service

import (
	"context"
	"errors"
	"time"

	"rag-iishka/internal/dto"
	"rag-iishka/internal/models"
	"rag-iishka/internal/repository"
	"rag-iishka/pkg/auth"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
)

type AuthService struct {
	userRepo   *repository.UserRepository
	jwtManager *auth.JWTManager
	logger     *zap.Logger
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *auth.JWTManager, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	// Check if user exists
	existingUser, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	now := time.Now()
	user := &models.User{
		ID:        uuid.New(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateToken(user.ID.String(), user.Username, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtManager.GetTokenDuration().Seconds()),
		User: dto.UserResponse{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
		},
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !auth.CheckPasswordHash(req.Password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateToken(user.ID.String(), user.Username, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtManager.GetTokenDuration().Seconds()),
		User: dto.UserResponse{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
		},
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*dto.AuthResponse, error) {
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Generate new tokens
	accessToken, err := s.jwtManager.GenerateToken(user.ID.String(), user.Username, user.Email)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtManager.GetTokenDuration().Seconds()),
		User: dto.UserResponse{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
		},
	}, nil
}
