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

type StoreController struct {
	DB *gorm.DB
}

func NewStoreController(db *gorm.DB) *StoreController {
	return &StoreController{DB: db}
}

// Request structs
type CreateStoreRequest struct {
	StoreCode string `json:"storeCode" validate:"required,min=3,max=50"`
	StoreName string `json:"storeName" validate:"required,min=3,max=100"`
}

type UpdateStoreRequest struct {
	StoreCode string `json:"storeCode" validate:"required,min=3,max=50"`
	StoreName string `json:"storeName" validate:"required,min=3,max=100"`
}

// GetStores retrieves a list of stores with pagination and search
// @Summary Get Stores
// @Description Retrieve a list of stores with pagination and search
// @Tags Stores
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of stores per page" default(10)
// @Param search query string false "Search term for store code or name"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.Store}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/stores [get]
func (bc *StoreController) GetStores(c fiber.Ctx) error {
	log.Println("GetStores called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var stores []models.Store

	// Build base query
	query := bc.DB.Model(&models.Store{}).Order("created_at DESC")

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("store_code ILIKE ? OR store_name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Limit(limit).Offset(offset).Find(&stores).Error; err != nil {
		log.Println("GetStores - Failed to retrieve stores:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data toko",
		})
	}

	// Format response
	storeList := make([]models.StoreResponse, len(stores))
	for i, store := range stores {
		storeList[i] = *store.ToResponse()
	}

	// Build success message
	message := "Stores retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println("GetStores completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    storeList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetStore retrieves a single store by ID
// @Summary Get Store
// @Description Retrieve a single store by ID
// @Tags Stores
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Store ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Store}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/stores/{id} [get]
func (bc *StoreController) GetStore(c fiber.Ctx) error {
	log.Println("GetStore called")
	// Parse id parameter
	id := c.Params("id")
	var store models.Store
	if err := bc.DB.Where("id = ?", id).First(&store).Error; err != nil {
		log.Println("GetStore - Store not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Toko dengan id " + id + " tidak ditemukan.",
		})
	}

	log.Println("GetStore completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data toko berhasil diambil",
		Data:    store.ToResponse(),
	})
}

// CreateStore creates a new store
// @Summary Create Store
// @Description Create a new store
// @Tags Stores
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param store body CreateStoreRequest true "Store details"
// @Success 201 {object} utils.SuccessResponse{data=models.Store}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/stores [post]
func (bc *StoreController) CreateStore(c fiber.Ctx) error {
	log.Println("CreateStore called")
	// Binding request body
	var req CreateStoreRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("CreateStore - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi data permintaan tidak valid",
		})
	}

	// Convert store code to uppercase and trim spaces
	req.StoreCode = strings.ToUpper(strings.TrimSpace(req.StoreCode))

	// Check for existing store with same code
	var existingStore models.Store
	if err := bc.DB.Where("store_code = ?", req.StoreCode).First(&existingStore).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Toko dengan kode " + req.StoreCode + " sudah terdaftar.",
		})
	}

	// Create new store
	newStore := models.Store{
		StoreCode: req.StoreCode,
		StoreName: req.StoreName,
	}

	if err := bc.DB.Create(&newStore).Error; err != nil {
		log.Println("CreateStore - Failed to create store:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat data toko",
		})
	}

	log.Println("CreateStore completed successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Toko berhasil tambahkan",
		Data:    newStore.ToResponse(),
	})
}

// UpdateStore updates an existing store by ID
// @Summary Update Store
// @Description Update an existing store by ID
// @Tags Stores
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Store ID"
// @Param request body UpdateStoreRequest true "Updated store details"
// @Success 200 {object} utils.SuccessResponse{data=models.Store}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/stores/{id} [put]
func (bc *StoreController) UpdateStore(c fiber.Ctx) error {
	log.Println("UpdateStore called")
	// Parse id parameter
	id := c.Params("id")
	var store models.Store
	if err := bc.DB.Where("id = ?", id).First(&store).Error; err != nil {
		log.Println("UpdateStore - Store not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Toko dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdateStoreRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Convert store code to uppercase and trim spaces
	req.StoreCode = strings.ToUpper(strings.TrimSpace(req.StoreCode))

	// Check for existing store with same code (excluding current store)
	var existingStore models.Store
	if err := bc.DB.Where("store_code = ? AND id != ?", req.StoreCode, id).First(&existingStore).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Toko dengan code " + req.StoreCode + " sudah terdaftar.",
		})
	}

	// Update store fields
	store.StoreCode = req.StoreCode
	store.StoreName = req.StoreName

	if err := bc.DB.Save(&store).Error; err != nil {
		log.Println("UpdateStore - Failed to update store:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui data toko",
		})
	}

	log.Println("UpdateStore completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Berhasil memperbarui data toko",
		Data:    store.ToResponse(),
	})
}

// DeleteStore deletes a store by ID
// @Summary Delete Store
// @Description Delete a store by ID
// @Tags Stores
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Store ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/stores/{id} [delete]
func (bc *StoreController) DeleteStore(c fiber.Ctx) error {
	log.Println("DeleteStore called")
	// Parse id parameter
	id := c.Params("id")
	var store models.Store
	if err := bc.DB.Where("id = ?", id).First(&store).Error; err != nil {
		log.Println("DeleteStore - Store not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Toko dengan id " + id + " tidak ditemukan.",
		})
	}

	// Delete store (also deletes associated records if any due to foreign key constraints)
	if err := bc.DB.Delete(&store).Error; err != nil {
		log.Println("DeleteStore - Failed to delete store:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghapus data toko",
		})
	}

	log.Println("DeleteStore completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data toko berhasil dihapus",
	})
}
