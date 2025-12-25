package handlers

import (
	"rag-iishka/internal/dto"
	"rag-iishka/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService *service.AuthService
	logger      *zap.Logger
}

func NewAuthHandler(authService *service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with username, email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration request"
// @Success 201 {object} dto.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /user/auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	resp, err := h.authService.Register(c.Context(), &req)
	if err != nil {
		if err == service.ErrUserExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "User already exists",
			})
		}
		h.logger.Error("Registration failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Registration failed",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// Login godoc
// @Summary Login user
// @Description Login with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login request"
// @Success 200 {object} dto.AuthResponse
// @Failure 401 {object} map[string]string
// @Router /user/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	resp, err := h.authService.Login(c.Context(), &req)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}
		h.logger.Error("Login failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Login failed",
		})
	}

	return c.JSON(resp)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Refresh access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} dto.AuthResponse
// @Failure 401 {object} map[string]string
// @Router /user/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req dto.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	resp, err := h.authService.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid refresh token",
			})
		}
		h.logger.Error("Token refresh failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Token refresh failed",
		})
	}

	return c.JSON(resp)
}

