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

type LostFoundController struct {
	DB *gorm.DB
}

func NewLostFoundController(db *gorm.DB) *LostFoundController {
	return &LostFoundController{DB: db}
}

// Request structs
type CreateLostFoundRequest struct {
	ProductSKU string `json:"productSKU" validate:"required"`
	Quantity   int    `json:"quantity" validate:"required,min=1"`
	Reason     string `json:"reason" validate:"required,min=3,max=255"`
}

type UpdateLostFoundRequest struct {
	Quantity int    `json:"quantity" validate:"omitempty,min=1"`
	Reason   string `json:"reason" validate:"omitempty,min=3,max=255"`
}

// GetLostfounds retrieves a list of lost and found records with pagination and search
// @Summary Get Lost and Found Records
// @Description Retrieve a list of lost and found records with pagination and search
// @Tags LostAndFound
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of records per page" default(10)
// @Param search query string false "Search term for product SKU or reason"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.LostFoundResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/lostfounds [get]
func (lfc *LostFoundController) GetLostfounds(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var lostFounds []models.LostFound

	// Build base query
	query := lfc.DB.Model(&models.LostFound{}).Preload("CreateUser").Order("created_at DESC")

	// Search condition if provided
	search := c.Query("search", "")
	if search != "" {
		query = query.Where("product_sku ILIKE ? OR reason ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated records
	if err := query.Offset(offset).Limit(limit).Find(&lostFounds).Error; err != nil {
		log.Println("Error retrieving lost and found records:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data barang hilang dan ditemukan",
		})
	}

	// load product details for each lost and found
	for i, lf := range lostFounds {
		var product models.Product
		if err := lfc.DB.Where("sku = ?", lf.ProductSKU).First(&product).Error; err == nil {
			lostFounds[i].Product = &product
		}
	}

	// Format response
	lostFoundList := make([]models.LostFoundResponse, len(lostFounds))
	for i, lostFound := range lostFounds {
		lostFoundList[i] = lostFound.ToResponse()
	}

	// Build success message
	message := "Lost and founds retrieved successfully"
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
		Data:    lostFoundList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetLostfound retrieves a single lost and found record by ID
// @Summary Get Lost and Found Record
// @Description Retrieve a single lost and found record by ID
// @Tags LostAndFound
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lost and Found ID"
// @Success 200 {object} utils.SuccessResponse{data=models.LostFoundResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/lostfounds/{id} [get]
func (lfc *LostFoundController) GetLostfound(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var lostFound models.LostFound
	if err := lfc.DB.Preload("CreateUser").Where("id = ?", id).First(&lostFound).Error; err != nil {
		log.Println("Lost and found record with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Data barang hilang dan ditemukan dengan id " + id + " tidak ditemukan.",
		})
	}

	log.Println("Lost and found record retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data barang hilang dan ditemukan berhasil diambil",
		Data:    lostFound.ToResponse(),
	})
}

// CreateLostfound creates a new lost and found record
// @Summary Create Lost and Found Record
// @Description Create a new lost and found record
// @Tags LostAndFound
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateLostFoundRequest true "Request details"
// @Success 201 {object} utils.SuccessResponse{data=models.LostFound}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/lostfounds [post]
func (lfc *LostFoundController) CreateLostfound(c fiber.Ctx) error {
	// Binding request body
	var req CreateLostFoundRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Convert product SKU to uppercase and trim spaces
	req.ProductSKU = strings.ToUpper(strings.TrimSpace(req.ProductSKU))

	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("Invalid user ID:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Check if product SKU exists in products table
	var product models.Product
	if err := lfc.DB.Where("sku = ?", req.ProductSKU).First(&product).Error; err != nil {
		log.Println("Product with SKU " + req.ProductSKU + " does not exist.")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Produk dengan SKU " + req.ProductSKU + " tidak ditemukan.",
		})
	}

	// Create new lost and found
	newLostFound := models.LostFound{
		ProductSKU: req.ProductSKU,
		Quantity:   req.Quantity,
		CreatedBy:  uint(userID),
		Reason:     req.Reason,
	}

	if err := lfc.DB.Create(&newLostFound).Error; err != nil {
		log.Println("Failed to create lost and found:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat data barang hilang dan ditemukan",
		})
	}

	// reload user related data and product data
	lfc.DB.Preload("CreateUser").First(&newLostFound, newLostFound.ID)
	lfc.DB.Where("sku = ?", newLostFound.ProductSKU).First(&product)
	newLostFound.Product = &product

	log.Println("Lost and found created successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data barang hilang dan ditemukan berhasil dibuat",
		Data:    newLostFound.ToResponse(),
	})
}

// UpdateLostfound updates an existing lost and found record
// @Summary Update Lost and Found Record
// @Description Update an existing lost and found record
// @Tags LostAndFound
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lost and Found ID"
// @Param request body UpdateLostFoundRequest true "Update request details"
// @Success 200 {object} utils.SuccessResponse{data=models.LostFound}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/lostfounds/{id} [put]
func (lfc *LostFoundController) UpdateLostfound(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var lostFound models.LostFound
	if err := lfc.DB.Where("id = ?", id).First(&lostFound).Error; err != nil {
		log.Println("Lost and found record with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Data barang hilang dan ditemukan dengan ID " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdateLostFoundRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Update fields if provided
	lostFound.Quantity = req.Quantity
	lostFound.Reason = req.Reason

	if err := lfc.DB.Save(&lostFound).Error; err != nil {
		log.Println("Failed to update lost and found:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui data barang hilang dan ditemukan",
		})
	}

	// reload user related data and product data
	lfc.DB.Preload("CreateUser").First(&lostFound, lostFound.ID)
	var product models.Product
	lfc.DB.Where("sku = ?", lostFound.ProductSKU).First(&product)
	lostFound.Product = &product

	log.Println("Lost and found updated successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data barang hilang dan ditemukan berhasil diperbarui",
		Data:    lostFound.ToResponse(),
	})
}

// DeleteLostfound deletes a lost and found record by ID
// @Summary Delete Lost and Found Record
// @Description Delete a lost and found record by ID
// @Tags LostAndFound
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lost and Found ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/lostfounds/{id} [delete]
func (lfc *LostFoundController) DeleteLostfound(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var lostFound models.LostFound
	if err := lfc.DB.Where("id = ?", id).First(&lostFound).Error; err != nil {
		log.Println("Lost and found record with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Data barang hilang dan ditemukan dengan id " + id + " tidak ditemukan.",
		})
	}

	// Delete lost and found
	if err := lfc.DB.Delete(&lostFound).Error; err != nil {
		log.Println("Failed to delete lost and found:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghapus data barang hilang dan ditemukan",
		})
	}

	log.Println("Lost and found deleted successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data barang hilang dan ditemukan berhasil dihapus",
	})
}
