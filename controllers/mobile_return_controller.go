package controllers

import (
	"fmt"
	"livo-fiber-backend/models"
	"livo-fiber-backend/utils"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type MobileReturnController struct {
	DB *gorm.DB
}

func NewMobileReturnController(db *gorm.DB) *MobileReturnController {
	return &MobileReturnController{DB: db}
}

// Request structs
type CreateMobileReturnRequest struct {
	NewTrackingNumber string `json:"newTrackingNumber" validate:"required"`
	ChannelID         uint   `json:"channelId" validate:"required"`
	StoreID           uint   `json:"storeId" validate:"required"`
}

// GetMobileReturns retrieves all mobile returns from the database
// @Summary Get Mobile Returns
// @Description Retrieve all mobile returns from the database
// @Tags Mobile Returns
// @Accept json
// @Produce json
// @Param page query int false "Page number for pagination" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param search query string false "Search term to filter returns by tracking number"
// @Success 200 {object} utils.SuccessTotaledResponse{data=[]models.ReturnResponse}
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-returns [get]
func (mrc *MobileReturnController) GetMobileReturns(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var mobileReturns []models.Return

	// Set date range: 7 days ago to current date
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	// Format dates for query
	startDateFormatted := startDate.Format("2006-01-02")
	endDateFormatted := endDate.Format("2006-01-02")

	// Build base query
	query := mrc.DB.Model(&models.Return{}).Preload("Channel").Preload("Store").Preload("CreateUser").Where("created_at >= ? AND created_at <= ?", startDateFormatted, endDateFormatted)

	// Parse search query parameter
	search := c.Query("search")
	if search != "" {
		query = query.Where("tracking_number ILIKE ?", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Execute the query to fetch mobile returns
	if err := query.Offset(offset).Limit(limit).Find(&mobileReturns).Error; err != nil {
		log.Println("Error retrieving mobile returns:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve mobile returns")
	}

	// Response format
	mobileReturnList := make([]models.ReturnResponse, len(mobileReturns))
	for i, ret := range mobileReturns {
		mobileReturnList[i] = ret.ToResponse()
	}

	// Build success message
	message := "Returns retrieved successfully"
	var filters []string

	// Add date range to filters
	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")
	filters = append(filters, fmt.Sprintf("date: from %s to %s", startDateStr, endDateStr))

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println(message)
	return c.Status(fiber.StatusOK).JSON(utils.SuccessTotaledResponse{
		Success: true,
		Message: message,
		Data:    mobileReturnList,
		Total:   total,
	})
}

// GetMobileReturn retrieves mobile returns by ID
// @Summary Get Mobile Return by ID
// @Description Retrieve mobile return by ID
// @Tags Mobile Returns
// @Accept json
// @Produce json
// @Param id path int true "Mobile Return ID"
// @Success 200 {object} utils.SuccessResponse{data=models.ReturnResponse}
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-returns/{id} [get]
func (mrc *MobileReturnController) GetMobileReturn(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var mobileReturn models.Return
	if err := mrc.DB.Preload("Channel").Preload("Store").Preload("CreateUser").Where("id = ?", id).First(&mobileReturn).Error; err != nil {
		log.Println("Mobile Return with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Mobile Return dengan id " + id + " tidak ditemukan.",
		})
	}

	log.Println("Mobile Return retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data mobile return berhasil diambil",
		Data:    mobileReturn.ToResponse(),
	})
}

// CreateMobileReturn creates a new mobile return
// @Summary Create Mobile Return
// @Description Create a new mobile return
// @Tags Mobile Returns
// @Accept json
// @Produce json
// @Param return body CreateMobileReturnRequest true "Mobile Return details"
// @Success 201 {object} utils.SuccessResponse{data=models.ReturnResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-returns [post]
func (mrc *MobileReturnController) CreateMobileReturn(c fiber.Ctx) error {
	// Parse request body
	var req CreateMobileReturnRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check existing return with same new tracking number
	var existingReturn models.Return
	if err := mrc.DB.Where("new_tracking_number = ?", req.NewTrackingNumber).First(&existingReturn).Error; err == nil {
		log.Println("Return with new tracking number " + req.NewTrackingNumber + " already exists.")
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Return dengan nomor pelacak baru " + req.NewTrackingNumber + " sudah terdaftar.",
		})
	}

	// Create new mobile return
	mobileReturn := models.Return{
		NewTrackingNumber: req.NewTrackingNumber,
		ChannelID:         req.ChannelID,
		StoreID:           req.StoreID,
		CreatedBy:         2,
	}

	if err := mrc.DB.Create(&mobileReturn).Error; err != nil {
		log.Println("Failed to create mobile return:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat  mobile return",
		})
	}

	// Load related data for response
	if err := mrc.DB.Preload("Channel").Preload("Store").Preload("CreateUser").Where("id = ?", mobileReturn.ID).First(&mobileReturn).Error; err != nil {
		log.Println("Failed to retrieve created mobile return:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data mobile return yang baru dibuat",
		})
	}

	log.Println("Mobile Return created successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Mobile return berhasil dibuat",
		Data:    mobileReturn.ToMobileResponse(),
	})
}
