package controllers

import (
	"fmt"
	"livo-fiber-backend/models"
	"livo-fiber-backend/utils"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type UserController struct {
	DB *gorm.DB
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

// Request structs
type UpdateUserRequest struct {
	FullName string `json:"fullName" validate:"omitempty,min=3,max=100" example:"John Doe"`
	Email    string `json:"email" validate:"omitempty,email" example:"john@example.com"`
	IsActive *bool  `json:"isActive" validate:"omitempty" example:"true"`
}

type UpdatePasswordRequest struct {
	NewPassword        string `json:"newPassword" validate:"required,min=8" example:"SecurePass123"`
	ConfirmNewPassword string `json:"confirmNewPassword" validate:"required,eqfield=NewPassword" example:"SecurePass123"`
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50" example:"john_doe"`
	Password string `json:"password" validate:"required,min=8" example:"SecurePass123"`
	FullName string `json:"fullName" validate:"required,min=3,max=100" example:"John Doe"`
	Email    string `json:"email" validate:"required,email" example:"john@example.com"`
	RoleName string `json:"roleName,omitempty" example:"guest"` // Optional role assignment
}

type AssignRoleRequest struct {
	RoleName string `json:"roleName" validate:"required" example:"guest"`
}

type RemoveRoleRequest struct {
	RoleName string `json:"roleName" validate:"required" example:"guest"`
}

// GetUsers retrieves a paginated list of users with optional search and role filtering
// @Summary Get Users
// @Description Retrieve a paginated list of users with optional search and role filtering
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of users per page" default(10)
// @Param search query string false "Search term for username or full name"
// @Param role query string false "Filter users by role name"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.UserResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users [get]
func (uc *UserController) GetUsers(c fiber.Ctx) error {
	log.Println("GetUsers called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var users []models.User

	// Build base query
	query := uc.DB.Model(&models.User{}).Order("created_at DESC").Preload("Roles")

	// Filter by role if provided
	roleName := strings.TrimSpace(c.Query("roleName", ""))
	if roleName != "" {
		query = query.Joins("JOIN user_roles ON users.id = user_roles.user_id").
			Joins("JOIN roles ON user_roles.role_id = roles.id").
			Where("roles.role_name = ?", roleName)
	}

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("username ILIKE ? OR full_name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated users
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		log.Println("GetUsers - Failed to retrieve users:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data pengguna",
		})
	}

	// Format response
	userList := make([]models.UserResponse, len(users))
	for i, user := range users {
		userList[i] = *user.ToResponse()
	}

	// Build success message
	message := "Users retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println("GetUsers completed successfully")
	return c.JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    userList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetUser retrieves a single user by ID
// @Summary Get User
// @Description Retrieve a single user by ID
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utils.SuccessResponse{data=models.UserResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users/{id} [get]
func (uc *UserController) GetUser(c fiber.Ctx) error {
	log.Println("GetUser called")
	// Parse id parameter
	id := c.Params("id")
	var user models.User
	if err := uc.DB.Preload("Roles").Where("id = ?", id).First(&user).Error; err != nil {
		log.Println("GetUser - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna dengan id " + id + " tidak ditemukan.",
		})
	}

	log.Println("GetUser completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data pengguna berhasil diambil",
		Data:    user.ToResponse(),
	})
}

// CreateUser creates a new user with optional role assignment
// @Summary Create User
// @Description Create a new user with optional role assignment
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserRequest true "User details"
// @Success 201 {object} utils.SuccessResponse{data=models.UserResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users [post]
func (uc *UserController) CreateUser(c fiber.Ctx) error {
	log.Println("CreateUser called")
	// Binding request body
	var req CreateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("CreateUser - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi data permintaan tidak valid",
		})
	}

	// Check for existing username or email
	var existingUser models.User
	if err := uc.DB.Preload("Roles").Where("username = ?", req.Username).Or("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Username atau email sudah terdaftar",
		})
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengenkripsi kata sandi",
		})
	}

	// Start database transaction
	tx := uc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create user
	newUser := models.User{
		Username: req.Username,
		Password: hashedPassword,
		FullName: req.FullName,
		Email:    req.Email,
		IsActive: true,
	}

	if err := tx.Create(&newUser).Error; err != nil {
		log.Println("CreateUser - Failed to create user:", err)
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menambahkan data pengguna",
		})
	}

	// Assign role if provided
	if req.RoleName != "" {
		var role models.Role
		if err := tx.Where("role_name = ?", req.RoleName).First(&role).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Nama role tidak valid",
			})
		}

		// Check permission hierarchy - current user must have higher or equal privilege
		currUserRoles := c.Locals("userRoles").([]string)
		currUserMinHierarchy := 999
		for _, currUserRoleName := range currUserRoles {
			var currRole models.Role
			if err := tx.Where("role_name = ?", currUserRoleName).First(&currRole).Error; err == nil {
				if currRole.Hierarchy < currUserMinHierarchy {
					currUserMinHierarchy = currRole.Hierarchy
				}
			}
		}

		// Current user must have equal or higher privilege (lower or equal hierarchy number)
		if role.Hierarchy < currUserMinHierarchy {
			tx.Rollback()
			return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Tidak punya hak akses",
			})
		}

		// Assign role to user
		userRole := models.UserRole{
			UserID: newUser.ID,
			RoleID: role.ID,
		}

		if err := tx.Create(&userRole).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Gagal memberikan role pada pengguna",
			})
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menambahkan pengguna",
		})
	}

	// Reload the data
	if err := uc.DB.Preload("Roles").Where("id = ?", newUser.ID).First(&newUser).Error; err != nil {
		log.Println("CreateUser - Failed to load user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat data pengguna",
		})
	}

	log.Println("CreateUser completed successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil dibuat",
		Data:    newUser.ToResponse(),
	})
}

// UpdateUser updates user details
// @Summary Update User
// @Description Update user details
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body UpdateUserRequest true "Updated user details"
// @Success 200 {object} utils.SuccessResponse{data=models.UserResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users/{id} [put]
func (uc *UserController) UpdateUser(c fiber.Ctx) error {
	log.Println("UpdateUser called")
	// Parse id parameter
	id := c.Params("id")
	var user models.User
	if err := uc.DB.Preload("Roles").Where("id = ?", id).First(&user).Error; err != nil {
		log.Println("UpdateUser - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Users can only update their own profile unless they have developer/superadmin/hrd role
	currUserID := c.Locals("userId").(string)
	if id != currUserID {
		if !utils.HasPermission(c, []string{"developer", "superadmin", "hrd"}) {
			return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Tidak memiliki hak akses untuk memperbarui profil pengguna lain",
			})
		}
	}

	// Update fields if provided
	user.FullName = req.FullName
	// If updating email, check for uniqueness
	if req.Email != "" && req.Email != user.Email {
		var existingUser models.User
		if err := uc.DB.Where("email = ? AND id != ?", req.Email, id).First(&existingUser).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Email sudah digunakan",
			})
		}
		user.Email = req.Email
	}
	// Only developer/superadmin/hrd can update IsActive
	if req.IsActive != nil {
		if !utils.HasPermission(c, []string{"developer", "superadmin", "hrd"}) {
			return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Pengguna tidak mempunyai hak akses",
			})
		}
		user.IsActive = *req.IsActive
	}

	if err := uc.DB.Save(&user).Error; err != nil {
		log.Println("UpdateUser - Failed to update user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui data pengguna",
		})
	}

	// Reload the data with fresh query
	var reloadedUser models.User
	if err := uc.DB.Preload("Roles").Where("id = ?", user.ID).First(&reloadedUser).Error; err != nil {
		log.Println("UpdateUser - Failed to load user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat data pengguna",
		})
	}

	log.Println("UpdateUser completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil diperbarui",
		Data:    reloadedUser.ToResponse(),
	})
}

// UpdatePassword updates a user's password
// @Summary Update Password
// @Description Update a user's password
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body UpdatePasswordRequest true "Updated password details"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users/{id}/password [put]
func (uc *UserController) UpdatePassword(c fiber.Ctx) error {
	log.Println("UpdatePassword called")
	// Parse id parameter
	id := c.Params("id")
	var user models.User
	if err := uc.DB.Where("id = ?", id).First(&user).Error; err != nil {
		log.Println("UpdatePassword - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdatePasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi data permintaan tidak valid",
		})
	}

	// Users can only update their own password unless they have developer/superadmin/hrd role
	currUserID := c.Locals("userId").(string)
	if id != currUserID {
		if !utils.HasPermission(c, []string{"developer", "superadmin", "hrd"}) {
			return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Tidak memiliki hak akses untuk mengubah password pengguna lain",
			})
		}
	}

	// Check if new password and confirm password match
	if req.NewPassword != req.ConfirmNewPassword {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Password baru dan konfirmasi password tidak sama",
		})
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengenkripsi password",
		})
	}

	user.Password = hashedPassword
	if err := uc.DB.Save(&user).Error; err != nil {
		log.Println("UpdatePassword - Failed to update password:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui password",
		})
	}

	// Clear sessions or tokens if user updated their own password
	uc.DB.Where("user_id = ?", user.ID).Delete(&models.Session{})

	log.Println("UpdatePassword completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Berhasil memperbarui password",
	})
}

// DeleteUser deletes a user by ID and all associated sessions
// @Summary Delete User
// @Description Delete a user by ID and all associated sessions
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users/{id} [delete]
func (uc *UserController) DeleteUser(c fiber.Ctx) error {
	log.Println("DeleteUser called")
	// Parse id parameter
	id := c.Params("id")
	var user models.User
	if err := uc.DB.Where("id = ?", id).First(&user).Error; err != nil {
		log.Println("DeleteUser - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna dengan id " + id + " tidak ditemukan.",
		})
	}

	// Delete all sessions associated with the user
	uc.DB.Where("user_id = ?", user.ID).Delete(&models.Session{})

	// Delete user (also deletes user_roles due to foreign key constraint with ON DELETE CASCADE)
	if err := uc.DB.Delete(&user).Error; err != nil {
		log.Println("DeleteUser - Failed to delete user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghapus pengguna",
		})
	}

	log.Println("DeleteUser completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil dihapus",
	})
}

// AssignRole assign a role to a user
// @Summary Assign Role
// @Description Assign a role to a user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body AssignRoleRequest true "Role to assign"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users/{id}/roles [post]
func (uc *UserController) AssignRole(c fiber.Ctx) error {
	log.Println("AssignRole called")
	// Parse id parameter
	id := c.Params("id")
	var user models.User
	if err := uc.DB.Where("id = ?", id).First(&user).Error; err != nil {
		log.Println("AssignRole - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "User dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req AssignRoleRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi data permintaan tidak valid",
		})
	}

	// Get the role to assign
	var role models.Role
	if err := uc.DB.Where("role_name = ?", req.RoleName).First(&role).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Nama role tidak valid",
		})
	}

	// Check if user already has the role
	var userRole models.UserRole
	if err := uc.DB.Where("user_id = ? AND role_id = ?", user.ID, role.ID).First(&userRole).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna sudah memiliki role tersebut",
		})
	}

	// Check permission hierarchy - current user must have higher or equal privilege
	currUserRoles := c.Locals("userRoles").([]string)
	currUserMinHierarchy := 999
	for _, currUserRoleName := range currUserRoles {
		var currRole models.Role
		if err := uc.DB.Where("role_name = ?", currUserRoleName).First(&currRole).Error; err == nil {
			if currRole.Hierarchy < currUserMinHierarchy {
				currUserMinHierarchy = currRole.Hierarchy
			}
		}
	}

	// Current user must have equal or higher privilege (lower or equal hierarchy number)
	if role.Hierarchy < currUserMinHierarchy {
		return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Izin tidak mencukupi",
		})
	}

	// Assign role to user
	newUserRole := models.UserRole{
		UserID: user.ID,
		RoleID: role.ID,
	}

	if err := uc.DB.Create(&newUserRole).Error; err != nil {
		log.Println("AssignRole - Failed to assign role to user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memberikan role ke pengguna",
		})
	}

	// Reload the data with fresh query
	var reloadedUser models.User
	if err := uc.DB.Preload("Roles").Where("id = ?", user.ID).First(&reloadedUser).Error; err != nil {
		log.Println("AssignRole - Failed to load user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat data pengguna",
		})
	}

	log.Println("AssignRole completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Role berhasil diberikan ke pengguna",
		Data:    reloadedUser.ToResponse(),
	})
}

// RemoveRole removes a role from a user
// @Summary Remove Role
// @Description Remove a role from a user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body RemoveRoleRequest true "Role to remove"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users/{id}/roles [delete]
func (uc *UserController) RemoveRole(c fiber.Ctx) error {
	log.Println("RemoveRole called")
	// Parse id parameter
	id := c.Params("id")
	var user models.User
	if err := uc.DB.Where("id = ?", id).First(&user).Error; err != nil {
		log.Println("RemoveRole - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req RemoveRoleRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan data tidak valid",
		})
	}

	// Get the role to remove
	var role models.Role
	if err := uc.DB.Where("role_name = ?", req.RoleName).First(&role).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Nama role tidak valid",
		})
	}

	// Check if user has the role
	var userRole models.UserRole
	if err := uc.DB.Where("user_id = ? AND role_id = ?", user.ID, role.ID).First(&userRole).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak memiliki role tersebut",
		})
	}

	// Check permission hierarchy - current user must have higher or equal privilege
	currUserRoles := c.Locals("userRoles").([]string)
	currUserMinHierarchy := 999
	for _, currUserRoleName := range currUserRoles {
		var currRole models.Role
		if err := uc.DB.Where("role_name = ?", currUserRoleName).First(&currRole).Error; err == nil {
			if currRole.Hierarchy < currUserMinHierarchy {
				currUserMinHierarchy = currRole.Hierarchy
			}
		}
	}

	// Current user must have equal or higher privilege (lower or equal hierarchy number)
	if role.Hierarchy < currUserMinHierarchy {
		return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Izin tidak mencukupi",
		})
	}

	// Remove role from user
	if err := uc.DB.Delete(&models.UserRole{}, "user_id = ? AND role_id = ?", user.ID, role.ID).Error; err != nil {
		log.Println("RemoveRole - Failed to remove role from user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghapus role dari pengguna",
		})
	}

	// Reload the data with fresh query
	var reloadedUser models.User
	if err := uc.DB.Preload("Roles").Where("id = ?", user.ID).First(&reloadedUser).Error; err != nil {
		log.Println("RemoveRole - Failed to load user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat data pengguna",
		})
	}

	log.Println("RemoveRole completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Role berhasil dihapus dari pengguna",
		Data:    reloadedUser.ToResponse(),
	})
}

// GetSessions retrieves all active sessions for a user
// @Summary Get User Sessions
// @Description Retrieve all active sessions for a user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utils.SuccessResponse{data=[]models.SessionResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users/{id}/sessions [get]
func (uc *UserController) GetSessions(c fiber.Ctx) error {
	log.Println("GetSessions called")
	// Parse id parameter
	id := c.Params("id")
	var user models.User
	if err := uc.DB.Where("id = ?", id).First(&user).Error; err != nil {
		log.Println("GetSessions - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna dengan id " + id + " tidak ditemukan.",
		})
	}

	// Users can only view their own sessions unless they have developer/superadmin/hrd role
	currUserID := c.Locals("userId").(string)
	if id != currUserID {
		if !utils.HasPermission(c, []string{"developer", "superadmin", "hrd"}) {
			return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Tidak memiliki hak akses untuk melihat session pengguna lain",
			})
		}
	}

	var sessions []models.Session
	if err := uc.DB.Where("user_id = ?", user.ID).Find(&sessions).Error; err != nil {
		log.Println("GetSessions - Failed to retrieve user sessions:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data sesi dari pengguna",
		})
	}

	// Format response
	sessionList := make([]models.SessionResponse, len(sessions))
	for i, session := range sessions {
		sessionList[i] = *session.ToResponse()
	}

	log.Println("GetSessions completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data sesi pengguna berhasil diambil",
		Data:    sessionList,
	})
}

// RegisterUserFace registers a new face for the user
// @Summary Register User Face
// @Description Register a new face for the user
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param image formData file true "Face image to register"
// @Success 201 {object} utils.SuccessResponse{data=models.UserFace}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/users/{id}/face-register [post]
func (uc *UserController) RegisterUserFace(c fiber.Ctx) error {
	log.Println("RegisterUserFace called")
	// Parse id parameter
	id := c.Params("id")
	userID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		log.Println("RegisterUserFace - Invalid user ID:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Check if user exists
	var user models.User
	if err := uc.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna dengan id " + id + " tidak ditemukan.",
		})
	}

	// Get uploaded image
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "File gambar wajib diunggah",
		})
	}

	// Validate mime type
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tipe gambar tidak valid",
		})
	}

	// Save temp file
	tmpPath := fmt.Sprintf("tmp/face_%d.jpg", userID)
	if err := c.SaveFile(file, tmpPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyimpan file gambar",
		})
	}
	defer os.Remove(tmpPath)

	// Call deepface service to register face
	if err := utils.SendToDeepFaceRegister(uint(userID), tmpPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Gagal mendaftarkan wajah ke layanan Deepface: %v", err),
		})
	}

	// Create or update user face record in database
	var userFace models.UserFace
	if err := uc.DB.Where("user_id = ?", userID).First(&userFace).Error; err != nil {
		userFace = models.UserFace{
			UserID:   uint(userID),
			IsActive: true,
		}
		if err := uc.DB.Create(&userFace).Error; err != nil {
			log.Println("RegisterUserFace - Failed to register user face:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Gagal mendaftarkan wajah pengguna",
			})
		}
	} else {
		userFace.IsActive = true
		if err := uc.DB.Save(&userFace).Error; err != nil {
			log.Println("RegisterUserFace - Failed to update user face:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Gagal memperbarui data wajah pengguna",
			})
		}
	}

	log.Println("RegisterUserFace completed successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Wajah pengguna berhasil ditambahkan",
		Data:    userFace,
	})
}
