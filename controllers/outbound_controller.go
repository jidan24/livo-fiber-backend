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

type OutboundController struct {
	DB *gorm.DB
}

func NewOutboundController(db *gorm.DB) *OutboundController {
	return &OutboundController{DB: db}
}

// Request structs
type CreateOutboundRequest struct {
	TrackingNumber  string `json:"trackingNumber" validate:"required,min=4,max=100"`
	Expedition      string `json:"expedition"`
	ExpeditionSlug  string `json:"expeditionSlug"`
	ExpeditionColor string `json:"expeditionColor"`
}

type UpdateOutboundRequest struct {
	Expedition      string `json:"expedition" validate:"required"`
	ExpeditionSlug  string `json:"expeditionSlug" validate:"required"`
	ExpeditionColor string `json:"expeditionColor" validate:"required"`
}

// Unique response structs
// OutboundsDailyCount represents the count of outbounds for a specific date
type OutboundsDailyCount struct {
	Date  time.Time `json:"date"`
	Count int64     `json:"count"`
}

// OutboundsDailyCountResponse represents the response for daily outbound counts
type OutboundsDailyCountResponse struct {
	Month       string                `json:"month"`
	Year        int                   `json:"year"`
	DailyCounts []OutboundsDailyCount `json:"dailyCounts"`
	TotalCount  int                   `json:"totalCount"`
}

// GetOutbounds retrieves a list of outbounds with pagination and search
// @Summary Get Outbounds
// @Description Retrieve a list of outbounds with pagination and search
// @Tags Outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of Outbounds per page" default(10)
// @Param search query string false "Search term for outbound Tracking Number"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.Outbound}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/outbounds [get]
func (oc *OutboundController) GetOutbounds(c fiber.Ctx) error {
	log.Println("GetOutbounds called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var outbounds []models.Outbound

	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Get start of current day (midnight)
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Build base query
	query := oc.DB.Model(&models.Outbound{}).Preload("OutboundUser").Order("created_at DESC").Where("outbound_by =?", uint(userID)).Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("tracking_number ILIKE ?", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&outbounds).Error; err != nil {
		log.Println("GetOutbounds - Failed to retrieve outbounds:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data outbounds",
		})
	}

	// Load orders for each QC Ribbon by tracking number
	for i := range outbounds {
		var order models.Order
		if err := oc.DB.Preload("OrderDetails").Where("tracking_number = ?", outbounds[i].TrackingNumber).First(&order).Error; err == nil {
			outbounds[i].Order = &order
		}
	}

	// Format response
	outboundList := make([]models.OutboundResponse, len(outbounds))
	for i, outbound := range outbounds {
		outboundList[i] = *outbound.ToResponse()
	}

	// Build success message
	message := "Outbounds retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println("GetOutbounds completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    outboundList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetOutbound retrieves a single outbound by ID
// @Summary Get Outbound
// @Description Retrieve a single outbound by ID
// @Tags Outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Outbound ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Outbound}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/outbounds/{id} [get]
func (oc *OutboundController) GetOutbound(c fiber.Ctx) error {
	log.Println("GetOutbound called")
	// Parse id parameter
	id := c.Params("id")
	var outbound models.Outbound
	if err := oc.DB.Preload("OutboundUser").Where("id = ?", id).First(&outbound).Error; err != nil {
		log.Println("GetOutbound - Outbound not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Outbound dengan id" + id + " tidak ditemukan.",
		})
	}

	// Load order by tracking number
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").Where("tracking_number = ?", outbound.TrackingNumber).First(&order).Error; err == nil {
		outbound.Order = &order
	}

	log.Println("GetOutbound completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data outbound berhasil diambil",
		Data:    outbound.ToResponse(),
	})
}

// CreateOutbound creates a new outbound
// @Summary Create Outbound
// @Description Create a new outbound
// @Tags Outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param outbound body CreateOutboundRequest true "Outbound data"
// @Success 201 {object} utils.SuccessResponse{data=models.Outbound}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/outbounds [post]
func (oc *OutboundController) CreateOutbound(c fiber.Ctx) error {
	log.Println("CreateOutbound called")
	// Binding request body
	var req CreateOutboundRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("CreateOutbound - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Convert tracking number to uppercase and trim spaces
	req.TrackingNumber = strings.ToUpper(strings.TrimSpace(req.TrackingNumber))

	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Check if tracking number exists in orders and processing status is "qc completed"
	var order models.Order
	if err := oc.DB.Where("tracking_number = ? AND processing_status = ?", req.TrackingNumber, "qc_completed").First(&order).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Nomor pelacakan tidak ada pada pesanan berstatus qc completed'",
		})
	}

	// Check if tracking number is exist in qc ribbons or qc onlines
	var qcRibbon models.QCRibbon
	var qcOnline models.QCOnline
	qcRibbonExists := oc.DB.Where("tracking_number = ?", req.TrackingNumber).First(&qcRibbon).Error == nil
	qcOnlineExists := oc.DB.Where("tracking_number = ?", req.TrackingNumber).First(&qcOnline).Error == nil

	// Tracking number must exist in either qc ribbons or qc onlines
	if !qcRibbonExists && !qcOnlineExists {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Nomor pelacakan harus ada di QC Ribbons atau QC Onlines",
		})
	}

	// Check if outbound tracking number already exists
	var existingOutbound models.Outbound
	if err := oc.DB.Where("tracking_number = ?", req.TrackingNumber).First(&existingOutbound).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Outbound dengan nomor pelacakan ini sudah ada",
		})
	}

	// Special case: If tracking number starts with "TKP0", use request body values
	var expedition, expeditionSlug, expeditionColor string

	if len(req.TrackingNumber) >= 4 && req.TrackingNumber[:4] == "TKP0" {
		expedition = req.Expedition
		expeditionColor = req.ExpeditionColor
		expeditionSlug = req.ExpeditionSlug
	} else {
		// Auto-detect expedition based on tracking prefix
		var expeditions []models.Expedition
		if err := oc.DB.Find(&expeditions).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Gagal mengambil data ekspedisi",
			})
		}

		expeditionFound := false

		// Check each expedition code to see if it matches the start of the tracking number
		for _, exp := range expeditions {
			if len(req.TrackingNumber) >= len(exp.ExpeditionCode) && req.TrackingNumber[:len(exp.ExpeditionCode)] == exp.ExpeditionCode {
				expedition = exp.ExpeditionName
				expeditionSlug = exp.ExpeditionSlug
				expeditionColor = exp.ExpeditionColor
				expeditionFound = true
				break
			}
		}

		// If no expedition found based on prefix, return error
		if !expeditionFound {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Tidak ditemukan ekspedisi untuk awalan nomor pelacakan yang diberikan",
			})
		}
	}

	outbound := models.Outbound{
		TrackingNumber:  req.TrackingNumber,
		OutboundBy:      uint(userID),
		Expedition:      expedition,
		ExpeditionSlug:  expeditionSlug,
		ExpeditionColor: expeditionColor,
	}

	if err := oc.DB.Create(&outbound).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat outbound",
		})
	}

	// Update order processing status to "outbound_completed" and event status to "completed"
	if err := oc.DB.Model(&models.Order{}).Where("tracking_number = ?", req.TrackingNumber).Update("processing_status", "outbound_completed").Update("event_status", "completed").Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pesanan",
		})
	}

	// reload created outbound with outbound user
	if err := oc.DB.Preload("OutboundUser").Where("id = ?", outbound.ID).First(&outbound).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data outbound yang baru dibuat",
		})
	}

	// load order by tracking number
	var orderResponse models.Order
	if err := oc.DB.Preload("OrderDetails").Where("tracking_number = ?", outbound.TrackingNumber).First(&orderResponse).Error; err == nil {
		outbound.Order = &orderResponse
	}

	log.Println("CreateOutbound completed successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Outbound berhasil dibuat",
		Data:    outbound.ToResponse(),
	})
}

// UpdateOutbound updates an existing outbound by ID (Only for tracking number starting with "TKP0")
// @Summary Update Outbound
// @Description Update an existing outbound by ID (Only for tracking number starting with "TKP0")
// @Tags Outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Outbound ID"
// @Param outbound body UpdateOutboundRequest true "Outbound data"
// @Success 200 {object} utils.SuccessResponse{data=models.Outbound}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/outbounds/{id} [put]
func (oc *OutboundController) UpdateOutbound(c fiber.Ctx) error {
	log.Println("UpdateOutbound called")
	// Parse id parameter
	id := c.Params("id")
	var outbound models.Outbound
	if err := oc.DB.Where("id = ?", id).First(&outbound).Error; err != nil {
		log.Println("UpdateOutbound - Outbound not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Outbound dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdateOutboundRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check if tracking number starts with "TKP0"
	if len(outbound.TrackingNumber) < 4 || outbound.TrackingNumber[:4] != "TKP0" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Hanya outbound dengan nomor pelacakan yang diawali 'TKPO' yang dapat diperbarui.",
		})
	}

	// Update outbound fields
	outbound.Expedition = req.Expedition
	outbound.ExpeditionSlug = req.ExpeditionSlug
	outbound.ExpeditionColor = req.ExpeditionColor

	if err := oc.DB.Save(&outbound).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui outbound",
		})
	}

	// reload updated outbound with outbound user
	if err := oc.DB.Preload("OutboundUser").Where("id = ?", outbound.ID).First(&outbound).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data outbound yang telah diperbarui",
		})
	}

	// load order by tracking number
	var orderResponse models.Order
	if err := oc.DB.Preload("OrderDetails").Where("tracking_number = ?", outbound.TrackingNumber).First(&orderResponse).Error; err == nil {
		outbound.Order = &orderResponse
	}

	log.Println("UpdateOutbound completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Outbound berhasil diperbarui",
		Data:    outbound.ToResponse(),
	})
}

// GetChartOutbounds retrieves daily outbound counts for the current month
// @Summary Get Outbound Chart Data
// @Description Retrieve daily outbound counts for the current month
// @Tags Outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.SuccessResponse{data=OutboundsDailyCountResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/outbounds/chart [get]
func (oc *OutboundController) GetChartOutbounds(c fiber.Ctx) error {
	log.Println("GetChartOutbounds called")
	// Get current month start and end dates
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	// Start of the month
	startOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

	// First day of next month at 00:00:00 (to use as upper bound)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	// Query to get daily counts for current month
	dailyCounts := []OutboundsDailyCount{}

	if err := oc.DB.Model(&models.Outbound{}).Select("DATE(created_at) as date, COUNT(*) as count").Where("created_at >= ? AND created_at < ?", startOfMonth, startOfNextMonth).Group("DATE(created_at)").Order("date ASC").Scan(&dailyCounts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data outbound",
		})
	}

	// Get total count for the current month
	var totalCount int64
	if err := oc.DB.Model(&models.Outbound{}).Where("created_at >= ? AND created_at < ?", startOfMonth, startOfNextMonth).Count(&totalCount).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil total jumlah outbound gagal",
		})
	}

	// Format response
	response := OutboundsDailyCountResponse{
		Month:       currentMonth.String(),
		Year:        currentYear,
		DailyCounts: dailyCounts,
		TotalCount:  int(totalCount),
	}

	message := "Outbound chart data " + currentMonth.String() + "  " + strconv.Itoa(currentYear) + " retrieved successfully"

	log.Println("GetChartOutbounds completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: message,
		Data:    response,
	})
}
