package controllers

import (
	"livo-fiber-backend/models"
	"livo-fiber-backend/utils"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type LocationController struct {
	DB *gorm.DB
}

func NewLocationController(db *gorm.DB) *LocationController {
	return &LocationController{DB: db}
}

// Request structs
type CreateLocationRequest struct {
	Name      string  `json:"name" validate:"required,min=3,max=100"`
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
}

type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
}

// GetLocations retrieves a list of locations with pagination and search
// @Summary Get Locations
// @Description Retrieve a list of locations with pagination and search
// @Tags Locations
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of locations per page" default(10)
// @Param search query string false "Search term for location name"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.LocationResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/locations [get]
func (lc *LocationController) GetLocations(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var locations []models.Location

	// Build base query
	query := lc.DB.Model(&models.Location{}).Order("created_at DESC")

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Offset(offset).Limit(limit).Find(&locations).Error; err != nil {
		log.Println("Error retrieving locations:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data lokasi",
		})
	}

	// Format locations for response
	locationList := make([]models.LocationResponse, len(locations))
	for i, loc := range locations {
		locationList[i] = *loc.ToResponse()
	}

	// Build success message
	message := "Locations retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += " with filters (" + strings.Join(filters, ", ") + ")"
	}

	// Return paginated response
	log.Println(message)
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    locationList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetLocation retrieves a single location by ID
// @Summary Get Location
// @Description Retrieve a single location by its ID
// @Tags Locations
// @Accept json
// @Produce json
// @Param id path int true "Location ID"
// @Success 200 {object} utils.SuccessResponse{data=models.LocationResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/locations/{id} [get]
func (lc *LocationController) GetLocation(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var location models.Location
	if err := lc.DB.Where("id = ?", id).First(&location).Error; err != nil {
		log.Println("Location not found")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Lokasi tidak ditemukan",
		})
	}

	log.Println("Location retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data lokasi berhasil diambil",
		Data:    location.ToResponse(),
	})
}

// CreateLocation creates a new location
// @Summary Create Location
// @Description Create a new location
// @Tags Locations
// @Accept json
// @Produce json
// @Param request body CreateLocationRequest true "Location data"
// @Success 201 {object} utils.SuccessResponse{data=models.LocationResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/locations [post]
func (lc *LocationController) CreateLocation(c fiber.Ctx) error {
	// Binding request body
	var req CreateLocationRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check for existing location with the same name
	var existing models.Location
	if err := lc.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		log.Println("Location with the same name already exists")
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Lokasi dengan nama yang sama sudah terdaftar",
		})
	}

	// Create new location
	location := models.Location{
		Name:      req.Name,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	if err := lc.DB.Create(&location).Error; err != nil {
		log.Println("Failed to create location:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat lokasi",
		})
	}

	log.Println("Location created successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Lokasi berhasil dibuat",
		Data:    location.ToResponse(),
	})
}

// UpdateLocation updates an existing location by ID
// @Summary Update Location
// @Description Update an existing location by its ID
// @Tags Locations
// @Accept json
// @Produce json
// @Param id path int true "Location ID"
// @Param request body UpdateLocationRequest true "Updated location data"
// @Success 200 {object} utils.SuccessResponse{data=models.LocationResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/locations/{id} [put]
func (lc *LocationController) UpdateLocation(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var location models.Location
	if err := lc.DB.Where("id = ?", id).First(&location).Error; err != nil {
		log.Println("Location not found")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Lokasi tidak ditemukan",
		})
	}

	// Binding request body
	var req UpdateLocationRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Update location fields
	location.Latitude = req.Latitude
	location.Longitude = req.Longitude

	if err := lc.DB.Save(&location).Error; err != nil {
		log.Println("Failed to update location:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui lokasi",
		})
	}

	log.Println("Location updated successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Berhasil memperbarui lokasi",
		Data:    location.ToResponse(),
	})
}

// DeleteLocation deletes a location by ID
// @Summary Delete Location
// @Description Delete a location by its ID
// @Tags Locations
// @Accept json
// @Produce json
// @Param id path int true "Location ID"
// @Success 204 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/locations/{id} [delete]
func (lc *LocationController) DeleteLocation(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var location models.Location
	if err := lc.DB.Where("id = ?", id).First(&location).Error; err != nil {
		log.Println("Location not found")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Lokasi tidak ditemukan",
		})
	}

	if err := lc.DB.Delete(&location).Error; err != nil {
		log.Println("Failed to delete location:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghapus lokasi",
		})
	}

	log.Println("Location deleted successfully")
	return c.Status(fiber.StatusNoContent).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Lokasi berhasil dihapus",
		Data:    nil,
	})
}
