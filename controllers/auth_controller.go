package controllers

import (
	"fmt"
	"livo-fiber-backend/config"
	"livo-fiber-backend/database"
	"livo-fiber-backend/models"
	"livo-fiber-backend/utils"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type AuthController struct {
	DB     *gorm.DB
	Config *config.Config
}

func NewAuthController(cfg *config.Config, db *gorm.DB) *AuthController {
	return &AuthController{Config: cfg, DB: db}
}

// Request structs
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50" example:"john_doe"`
	Password string `json:"password" validate:"required,min=8" example:"SecurePass123"`
	FullName string `json:"fullName" validate:"required,min=2,max=100" example:"John Doe"`
	Email    string `json:"email" validate:"required,email" example:"john@example.com"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required" example:"john_doe"`
	Password string `json:"password" validate:"required" example:"SecurePass123"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required" example:"v4.local.xxx"`
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account with username, password, full name, and email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} utils.SuccessResponse{data=models.UserResponse} "User registered successfully"
// @Failure 400 {object} utils.ErrorResponse "Invalid request body"
// @Failure 409 {object} utils.ErrorResponse "Username or email already exists"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /api/auth/register [post]
func (ac *AuthController) Register(c fiber.Ctx) error {
	var req RegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check if username exists
	var existingUser models.User
	if err := database.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		log.Println("Username already exists:", req.Username)
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Username sudah terdaftar",
		})
	}

	// Check if email exists
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		log.Println("Email already exists:", req.Email)
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Email sudah terdaftar",
		})
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Println("Failed to hash password:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal melakukan hash kata sandi",
		})
	}

	// Get guest role first to ensure it exists
	var guestRole models.Role
	if err := database.DB.Where("role_name = ?", "guest").First(&guestRole).Error; err != nil {
		log.Println("Failed to get default role:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil peran default",
		})
	}

	// Use transaction to ensure user and role assignment are created together
	var user models.User
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Create user
		user = models.User{
			Username: req.Username,
			Password: hashedPassword,
			FullName: req.FullName,
			Email:    req.Email,
			IsActive: true,
		}

		if err := tx.Create(&user).Error; err != nil {
			log.Panicln(err)
			return err
		}

		// Associate role with user
		if err := tx.Table("user_roles").Create(map[string]interface{}{
			"user_id": user.ID,
			"role_id": guestRole.ID,
		}).Error; err != nil {
			log.Panicln(err)
			return err
		}

		return nil
	})

	if err != nil {
		log.Println("Failed to create user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat pengguna",
		})
	}

	// Load roles for response
	database.DB.Preload("Roles").First(&user, "id = ?", user.ID)

	log.Println("User registered successfully:", req.Username)
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil terdaftar",
		Data:    user.ToResponse(),
	})
}

// Login handles user login
// @Summary User login
// @Description Authenticate user and return access token with optional refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} utils.LoginResponse "Login successful"
// @Failure 400 {object} utils.ErrorResponse "Invalid request body"
// @Failure 401 {object} utils.ErrorResponse "Invalid credentials"
// @Failure 403 {object} utils.ErrorResponse "User account is disabled"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /api/auth/login [post]
func (ac *AuthController) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Find user
	var user models.User
	if err := database.DB.Preload("Roles").Where("username = ?", req.Username).First(&user).Error; err != nil {
		log.Println("Invalid credentials for user:", req.Username)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Username/Katasandi tidak valid",
		})
	}

	// Check if user is active
	if !user.IsActive {
		log.Println("User account is disabled:", req.Username)
		return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Akun pengguna tidak aktif",
		})
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		log.Println("Invalid credentials for user:", req.Username)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Username/katasandi tidak valid",
		})
	}

	// Get role names
	roleNames := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roleNames[i] = role.RoleName
	}

	// Generate tokens
	claims := utils.TokenClaims{
		UserID:   fmt.Sprintf("%d", user.ID),
		Username: user.Username,
		Roles:    roleNames,
	}

	accessToken, err := utils.GenerateAccessToken(claims, ac.Config)
	if err != nil {
		log.Println("Failed to generate access token:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat akses token",
		})
	}

	refreshToken, err := utils.GenerateRefreshToken(claims, ac.Config)
	if err != nil {
		log.Println("Failed to generate refresh token:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagam membuat refresh token",
		})
	}

	// Detect device type
	userAgent := c.Get("User-Agent")
	deviceType := "web"

	// Check custom header first (for mobile apps like React Native/Expo)
	customDeviceType := c.Get("X-Device-Type")
	if customDeviceType != "" {
		if customDeviceType == "mobile" || customDeviceType == "ios" || customDeviceType == "android" {
			deviceType = "mobile"
		}
	} else if strings.Contains(strings.ToLower(userAgent), "mobile") {
		// Fallback to User-Agent detection
		deviceType = "mobile"
	}

	// Create session
	session := models.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		IPAddress:    c.IP(),
		DeviceType:   deviceType,
		ExpiresAt:    time.Now().Add(time.Duration(ac.Config.RefreshTokenTTL) * 24 * time.Hour),
	}

	if err := database.DB.Create(&session).Error; err != nil {
		log.Println("Failed to create session:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat sesi",
		})
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	database.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("last_login", now)

	userResponse := user.ToResponse()
	response := utils.LoginResponse{
		Success:     true,
		AccessToken: accessToken,
		User:        userResponse,
	}

	// If web app, set refresh token in httponly cookie
	if deviceType == "web" {
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
			MaxAge:   ac.Config.RefreshTokenTTL * 24 * 3600,
		})
	} else {
		// If mobile app, include refresh token in response
		response.RefreshToken = refreshToken
	}

	log.Println("User logged in successfully:", req.Username)
	return c.JSON(response)
}

// Logout handles user logout and clear current session
// @Summary User logout
// @Description Logout user and invalidate current session or all sessions
// @Tags Authentication
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <token>)
// @Param request body RefreshTokenRequest false "Optional refresh token to logout specific session"
// @Success 200 {object} utils.SuccessResponse "Logged out successfully"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /api/auth/logout [post]
func (ac *AuthController) Logout(c fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	// Get refresh token from cookie or body
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		var body struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := c.Bind().Body(&body); err == nil {
			refreshToken = body.RefreshToken
		}
	}

	if refreshToken != "" {
		// Delete specific session
		database.DB.Where("user_id = ? AND refresh_token = ?", userID, refreshToken).Delete(&models.Session{})
	} else {
		// Delete all sessions for user
		database.DB.Where("user_id = ?", userID).Delete(&models.Session{})
	}

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HTTPOnly: true,
		MaxAge:   -1,
	})

	log.Println("User logged out successfully, userID:", userID)
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Berhasil logout",
	})
}

// RefreshToken handles token refreshing and generating new access token
// @Summary Refresh access token
// @Description Generate a new access token using a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest false "Refresh token (optional if using cookie)"
// @Success 200 {object} utils.LoginResponse "Token refreshed successfully"
// @Failure 400 {object} utils.ErrorResponse "Refresh token required"
// @Failure 401 {object} utils.ErrorResponse "Invalid or expired refresh token"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /api/auth/refresh [post]
func (ac *AuthController) RefreshToken(c fiber.Ctx) error {
	// Get refresh token from cookie or body
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		var body struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := c.Bind().Body(&body); err != nil || body.RefreshToken == "" {
			log.Println("Refresh token required:", err)
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Refresh token diperlukan",
			})
		}
		refreshToken = body.RefreshToken
	}

	// Validate refresh token
	token, err := utils.ValidateToken(refreshToken, ac.Config)
	if err != nil {
		log.Println("Invalid or expired refresh token:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Refresh token tidak valid / kadaluwarsa",
		})
	}

	// Check token type
	tokenType, err := token.GetString("type")
	if err != nil || tokenType != "refresh" {
		log.Println("Invalid token type")
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Jenis token tidak valid",
		})
	}

	// Get session
	var session models.Session
	if err := database.DB.Preload("User.Roles").Where("refresh_token = ?", refreshToken).First(&session).Error; err != nil {
		log.Println("Session not found for refresh token:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Sesi tidak ditemukan",
		})
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		database.DB.Delete(&session)
		log.Println("Session expired for userID:", session.UserID)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Sesi telah kadaluwarsa",
		})
	}

	// Get role names
	roleNames := make([]string, len(session.User.Roles))
	for i, role := range session.User.Roles {
		roleNames[i] = role.RoleName
	}

	// Generate new access token
	claims := utils.TokenClaims{
		UserID:   fmt.Sprintf("%d", session.UserID),
		Username: session.User.Username,
		Roles:    roleNames,
	}

	newAccessToken, err := utils.GenerateAccessToken(claims, ac.Config)
	if err != nil {
		log.Println("Failed to generate access token:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghasilkan access token",
		})
	}

	// Optionally rotate refresh token
	newRefreshToken, err := utils.GenerateRefreshToken(claims, ac.Config)
	if err != nil {
		log.Println("Failed to generate refresh token:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghasilkan refresh token",
		})
	}

	// Update session
	session.RefreshToken = newRefreshToken
	session.ExpiresAt = time.Now().Add(time.Duration(ac.Config.RefreshTokenTTL) * 24 * time.Hour)
	database.DB.Save(&session)

	userResponse := session.User.ToResponse()
	response := utils.LoginResponse{
		Success:     true,
		AccessToken: newAccessToken,
		User:        userResponse,
	}

	// Update cookie for web or include in response for mobile
	if session.DeviceType == "web" {
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    newRefreshToken,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
			MaxAge:   ac.Config.RefreshTokenTTL * 24 * 3600,
		})
	} else {
		response.RefreshToken = newRefreshToken
	}

	log.Println("Token refreshed successfully for userID:", session.UserID)
	return c.JSON(response)
}
