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

type BoxController struct {
	DB *gorm.DB
}

func NewBoxController(db *gorm.DB) *BoxController {
	return &BoxController{DB: db}
}

// Request structs
type CreateBoxRequest struct {
	BoxCode string `json:"boxCode" validate:"required,min=3,max=50"`
	BoxName string `json:"boxName" validate:"required,min=3,max=100"`
}

type UpdateBoxRequest struct {
	BoxCode string `json:"boxCode" validate:"required,min=3,max=50"`
	BoxName string `json:"boxName" validate:"required,min=3,max=100"`
}

// GetBoxes retrieves a list of boxes with pagination and search
// @Summary Get Boxes
// @Description Retrieve a list of boxes with pagination and search
// @Tags Boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of boxes per page" default(10)
// @Param search query string false "Search term for box code or name"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.Box}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/boxes [get]
func (bc *BoxController) GetBoxes(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var boxes []models.Box

	// Build base query
	query := bc.DB.Model(&models.Box{}).Order("created_at DESC")

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("box_code ILIKE ? OR box_name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Limit(limit).Offset(offset).Find(&boxes).Error; err != nil {
		log.Println("Error retrieving boxes:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data box",
		})
	}

	// Format response
	boxList := make([]models.BoxResponse, len(boxes))
	for i, box := range boxes {
		boxList[i] = *box.ToResponse()
	}

	// Build success message
	message := "Boxes retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println(message)
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    boxList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetBox retrieves a single box by ID
// @Summary Get Box
// @Description Retrieve a single box by ID
// @Tags Boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Box ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Box}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/boxes/{id} [get]
func (bc *BoxController) GetBox(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var box models.Box
	if err := bc.DB.Where("id = ?", id).First(&box).Error; err != nil {
		log.Println("Box with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Box dengan id " + id + " tidak ditemukan.",
		})
	}

	log.Println("Box retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data box berhasil diambil",
		Data:    box.ToResponse(),
	})
}

// CreateBox creates a new box
// @Summary Create Box
// @Description Create a new box
// @Tags Boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param box body CreateBoxRequest true "Box details"
// @Success 201 {object} utils.SuccessResponse{data=models.Box}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/boxes [post]
func (bc *BoxController) CreateBox(c fiber.Ctx) error {
	// Binding request body
	var req CreateBoxRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Convert box code to uppercase and trim spaces
	req.BoxCode = strings.ToUpper(strings.TrimSpace(req.BoxCode))

	// Check for existing box with same code
	var existingBox models.Box
	if err := bc.DB.Where("box_code = ?", req.BoxCode).First(&existingBox).Error; err == nil {
		log.Println("Box with code " + req.BoxCode + " already exists.")
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Box dengan code " + req.BoxCode + " sudah terdaftar.",
		})
	}

	// Create new box
	newBox := models.Box{
		BoxCode: req.BoxCode,
		BoxName: req.BoxName,
	}

	if err := bc.DB.Create(&newBox).Error; err != nil {
		log.Println("Failed to create box:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat box",
		})
	}

	log.Println("Box created successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Box berhasil dibuat",
		Data:    newBox.ToResponse(),
	})
}

// UpdateBox updates an existing box by ID
// @Summary Update Box
// @Description Update an existing box by ID
// @Tags Boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Box ID"
// @Param request body UpdateBoxRequest true "Updated box details"
// @Success 200 {object} utils.SuccessResponse{data=models.Box}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/boxes/{id} [put]
func (bc *BoxController) UpdateBox(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var box models.Box
	if err := bc.DB.Where("id = ?", id).First(&box).Error; err != nil {
		log.Println("Box with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Box dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdateBoxRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Convert box code to uppercase and trim spaces
	req.BoxCode = strings.ToUpper(strings.TrimSpace(req.BoxCode))

	// Check for existing box with same code (excluding current box)
	var existingBox models.Box
	if err := bc.DB.Where("box_code = ? AND id != ?", req.BoxCode, id).First(&existingBox).Error; err == nil {
		log.Println("Box with code " + req.BoxCode + " already exists.")
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Box dengan code " + req.BoxCode + " sudah terdaftar.",
		})
	}

	// Update box fields
	box.BoxCode = req.BoxCode
	box.BoxName = req.BoxName

	if err := bc.DB.Save(&box).Error; err != nil {
		log.Println("Failed to update box:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui box",
		})
	}

	log.Println("Box updated successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Box berhasil diperbarui.",
		Data:    box.ToResponse(),
	})
}

// DeleteBox deletes a box by ID
// @Summary Delete Box
// @Description Delete a box by ID
// @Tags Boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Box ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/boxes/{id} [delete]
func (bc *BoxController) DeleteBox(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var box models.Box
	if err := bc.DB.Where("id = ?", id).First(&box).Error; err != nil {
		log.Println("Box with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Box dengan id " + id + " tidak ditemukan.",
		})
	}

	// Delete box (also deletes associated records if any due to foreign key constraints)
	if err := bc.DB.Delete(&box).Error; err != nil {
		log.Println("Failed to delete box:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghapus box",
		})
	}

	log.Println("Box deleted successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Box berhasil dihapus",
	})
}
