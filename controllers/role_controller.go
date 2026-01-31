package controllers

import (
	"fmt"
	"livo-fiber-backend/models"
	"livo-fiber-backend/utils"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type RoleController struct {
	DB *gorm.DB
}

func NewRoleController(db *gorm.DB) *RoleController {
	return &RoleController{DB: db}
}

// Request structs
type CreateRoleRequest struct {
	RoleName  string `json:"roleName" validate:"required,min=3,max=50"`
	Hierarchy int    `json:"hierarchy" validate:"required,min=1"`
}

type UpdateRoleRequest struct {
	RoleName  string `json:"roleName" validate:"required,min=3,max=50"`
	Hierarchy int    `json:"hierarchy" validate:"required,min=1"`
}

// GetRoles retrieves a list of roles with pagination and search
// @Summary Get Roles
// @Description Retrieve a list of roles with pagination and search
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of roles per page" default(10)
// @Param search query string false "Search term for role name"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.Role}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/roles [get]
func (rc *RoleController) GetRoles(c fiber.Ctx) error {
	log.Println("GetRoles called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var roles []models.Role

	// Build base query
	query := rc.DB.Model(&models.Role{}).Order("created_at DESC")

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("role_name ILIKE ?", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Limit(limit).Offset(offset).Find(&roles).Error; err != nil {
		log.Println("GetRoles - Failed to retrieve roles:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data peran",
		})
	}

	// Format response
	roleList := make([]models.RoleResponse, len(roles))
	for i, role := range roles {
		roleList[i] = *role.ToResponse()
	}

	// Build success message
	message := "Roles retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println("GetRoles completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    roleList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetRole retrieves a single role by ID
// @Summary Get Role
// @Description Retrieve a single role by ID
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Role}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/roles/{id} [get]
func (rc *RoleController) GetRole(c fiber.Ctx) error {
	log.Println("GetRole called")
	// Parse id parameter
	id := c.Params("id")
	var role models.Role
	if err := rc.DB.Where("id = ?", id).First(&role).Error; err != nil {
		log.Println("GetRole - Role not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Role dengan id " + id + " tidak ditemukan.",
		})
	}

	log.Println("GetRole completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Role berhasil diambil",
		Data:    role.ToResponse(),
	})
}

// CreateRole creates a new role
// @Summary Create Role
// @Description Create a new role
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRoleRequest true "Role details"
// @Success 201 {object} utils.SuccessResponse{data=models.Role}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/roles [post]
func (rc *RoleController) CreateRole(c fiber.Ctx) error {
	log.Println("CreateRole called")
	// Binding request body
	var req CreateRoleRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("CreateRole - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check for existing role with same name
	var existingRole models.Role
	if err := rc.DB.Where("role_name = ?", req.RoleName).First(&existingRole).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Role dengan nama " + req.RoleName + " sudah terdaftar.",
		})
	}

	// Check permission hierarchy - current user only can create roles with equal or lower hierarchy
	currUserRoles := c.Locals("userRoles").([]string)
	currUserMinHierarchy := 999
	for _, currUserRoleName := range currUserRoles {
		var currRole models.Role
		if err := rc.DB.Where("role_name = ?", currUserRoleName).First(&currRole).Error; err == nil {
			if currRole.Hierarchy < currUserMinHierarchy {
				currUserMinHierarchy = currRole.Hierarchy
			}
		}
	}

	// Validate new role hierarchy
	if req.Hierarchy < currUserMinHierarchy {
		return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Hak akses tidak cukup untuk membuat role dengan privilese lebih tinggi",
		})
	}

	// Create new role
	newRole := models.Role{
		RoleName:  req.RoleName,
		Hierarchy: req.Hierarchy,
	}

	if err := rc.DB.Create(&newRole).Error; err != nil {
		log.Println("CreateRole - Failed to create role:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat role",
		})
	}

	log.Println("CreateRole completed successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Role Berhasil dibuat",
		Data:    newRole.ToResponse(),
	})
}

// UpdateRole updates an existing role by ID
// @Summary Update Role
// @Description Update an existing role by ID
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Param request body UpdateRoleRequest true "Updated role details"
// @Success 200 {object} utils.SuccessResponse{data=models.Role}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/roles/{id} [put]
func (rc *RoleController) UpdateRole(c fiber.Ctx) error {
	log.Println("UpdateRole called")
	// Parse id parameter
	id := c.Params("id")
	var role models.Role
	if err := rc.DB.Where("id = ?", id).First(&role).Error; err != nil {
		log.Println("UpdateRole - Role not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Role dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdateRoleRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check for existing role with same name (excluding current role)
	var existingRole models.Role
	if err := rc.DB.Where("role_name = ? AND id != ?", req.RoleName, id).First(&existingRole).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Role dengan nama " + req.RoleName + " sudah terdaftar.",
		})
	}

	// Check permission hierarchy - current user only can create roles with equal or lower hierarchy
	currUserRoles := c.Locals("userRoles").([]string)
	currUserMinHierarchy := 999
	for _, currUserRoleName := range currUserRoles {
		var currRole models.Role
		if err := rc.DB.Where("role_name = ?", currUserRoleName).First(&currRole).Error; err == nil {
			if currRole.Hierarchy < currUserMinHierarchy {
				currUserMinHierarchy = currRole.Hierarchy
			}
		}
	}

	// Validate new role hierarchy
	if req.Hierarchy < currUserMinHierarchy {
		return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Hak akses tidak cukup untuk membuat role dengan privilies lebih tinggi",
		})
	}

	// Update role fields
	role.RoleName = req.RoleName
	role.Hierarchy = req.Hierarchy

	if err := rc.DB.Save(&role).Error; err != nil {
		log.Println("UpdateRole - Failed to update role:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui role",
		})
	}

	log.Println("UpdateRole completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Role berhasil diperbarui",
		Data:    role.ToResponse(),
	})
}

// DeleteRole deletes a role by ID
// @Summary Delete Role
// @Description Delete a role by ID
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/roles/{id} [delete]
func (rc *RoleController) DeleteRole(c fiber.Ctx) error {
	log.Println("DeleteRole called")
	// Parse id parameter
	id := c.Params("id")
	var role models.Role
	if err := rc.DB.Where("id = ?", id).First(&role).Error; err != nil {
		log.Println("DeleteRole - Role not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Role dengan id " + id + " tidak ditemukan.",
		})
	}

	// Delete role (also deletes user_roles due to foreign key constraint with ON DELETE CASCADE)
	if err := rc.DB.Delete(&role).Error; err != nil {
		log.Println("DeleteRole - Failed to delete role:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghapus role",
		})
	}

	log.Println("DeleteRole completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Role berhasil dihapus",
	})
}
