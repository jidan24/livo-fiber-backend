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

type QCRibbonController struct {
	DB *gorm.DB
}

func NewQCRibbonController(db *gorm.DB) *QCRibbonController {
	return &QCRibbonController{DB: db}
}

// Request structs
type QCRibbonStartRequest struct {
	TrackingNumber string `json:"trackingNumber" validate:"required"`
}

type ValidateQCRibbonProductRequest struct {
	SKU      string `json:"sku" validate:"required"`
	Quantity int    `json:"quantity" validate:"required,min=1"`
}

type CreateQCRibbonDetail struct {
	BoxID    uint `json:"boxId" validate:"required"`
	Quantity int  `json:"quantity" validate:"required,min=1"`
}

type CreateQCRibbonDetailRequest struct {
	Details []CreateQCRibbonDetail `json:"details" validate:"required,dive,required"`
}

// Unique response structs
// QcRibbonDailyCount represents the count of qc-ribbons for a specific date
type QcRibbonDailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// QcRibbonsDailyCountResponse represents the response for daily qc-ribbon counts
type QcRibbonsDailyCountResponse struct {
	Month       string               `json:"month"`
	Year        int                  `json:"year"`
	DailyCounts []QcRibbonDailyCount `json:"dailyCounts"`
	TotalCount  int                  `json:"totalCount"`
}

// GetQCRibbons retrieves a list of qc ribbons with pagination and search
// @Summary Get QC Ribbons
// @Description Retrieve a list of QC Ribbons with pagination and search
// @Tags Ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of QC Ribbons per page" default(10)
// @Param search query string false "Search term for tracking number"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.QCRibbonResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/ribbons/qc-ribbons [get]
func (qcrc *QCRibbonController) GetQCRibbons(c fiber.Ctx) error {
	log.Println("GetQCRibbons called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var qcRibbons []models.QCRibbon

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
	query := qcrc.DB.Model(&models.QCRibbon{}).Preload("QCRibbonDetails.Box").Preload("QCUser").Order("updated_at DESC").Where("qc_by = ?", uint(userID)).Where("updated_at >= ? AND updated_at < ?", startOfDay, endOfDay)

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("tracking_number ILIKE ?", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&qcRibbons).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data QC Ribbons",
		})
	}

	// Load orders for each QC Ribbon by tracking number
	for i := range qcRibbons {
		var order models.Order
		if err := qcrc.DB.Preload("OrderDetails").Where("tracking_number = ?", qcRibbons[i].TrackingNumber).First(&order).Error; err == nil {
			qcRibbons[i].Order = &order
		}
	}

	// Format response
	qcRibbonList := make([]models.QCRibbonResponse, len(qcRibbons))
	for i, qcRibbon := range qcRibbons {
		qcRibbonList[i] = *qcRibbon.ToResponse()
	}

	// Build success message
	message := "QC Ribbons retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println("GetQCRibbons completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    qcRibbonList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetQCRibbon retrieves a single qc ribbon by ID
// @Summary Get QC Ribbon
// @Description Retrieve a single QC Ribbon by ID
// @Tags Ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Ribbon ID"
// @Success 200 {object} utils.SuccessResponse{data=models.QCRibbonResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/ribbons/qc-ribbons/{id} [get]
func (qcrc *QCRibbonController) GetQCRibbon(c fiber.Ctx) error {
	log.Println("GetQCRibbon called")
	// Parse id parameter
	id := c.Params("id")
	var qcRibbon models.QCRibbon
	if err := qcrc.DB.Preload("QCRibbonDetails.Box").Preload("QCUser").Where("id = ?", id).First(&qcRibbon).Error; err != nil {
		log.Println("GetQCRibbon - QC Ribbon not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Ribbon dengan id " + id + " tidak ditemukan.",
		})
	}

	// Load order by tracking number
	var order models.Order
	if err := qcrc.DB.Preload("OrderDetails").Where("tracking_number = ?", qcRibbon.TrackingNumber).First(&order).Error; err == nil {
		qcRibbon.Order = &order
	}

	log.Println("GetQCRibbon completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "QC Ribbon berhasil diambil",
		Data:    qcRibbon.ToResponse(),
	})
}

// GetChartQcRibbons retrieves QC Ribbon data for charting
// @Summary Get Chart QC Ribbons
// @Description Retrieve QC Ribbon data for charting
// @Tags Ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.SuccessResponse{data=QcRibbonsDailyCountResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/ribbons/qc-ribbons/chart [get]
func (qcrc *QCRibbonController) GetChartQCRibbons(c fiber.Ctx) error {
	log.Println("GetChartQCRibbons called")
	// Get current month start and end dates
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	// Start of the month
	startOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

	// First day of next month at 00:00:00 (to use as upper bound)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	// Query to get daily counts for current month
	var dailyCounts []QcRibbonDailyCount

	if err := qcrc.DB.Model(&models.QCRibbon{}).Select("DATE(created_at) as date, COUNT(*) as count").Where("created_at >= ? AND created_at < ?", startOfMonth, startOfNextMonth).Group("DATE(created_at)").Order("date ASC").Scan(&dailyCounts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data QC Ribbon",
		})
	}

	// Get total count for the current month
	var totalCount int64
	if err := qcrc.DB.Model(&models.QCRibbon{}).Where("created_at >= ? AND created_at < ?", startOfMonth, startOfNextMonth).Count(&totalCount).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil total jumlah QC Ribbon",
		})
	}

	// Format response
	response := QcRibbonsDailyCountResponse{
		Month:       currentMonth.String(),
		Year:        currentYear,
		DailyCounts: dailyCounts,
		TotalCount:  int(totalCount),
	}

	message := "QC Ribbon chart data " + currentMonth.String() + "  " + strconv.Itoa(currentYear) + " retrieved successfully"

	log.Println("GetChartQCRibbons completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: message,
		Data:    response,
	})
}

// QCRibbonStart Starting QC Ribbon processing for an order
// @Summary Start QC Ribbon Processing
// @Description Mark an order as in QC Ribbon processing
// @Tags Ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param trackingNumber body QCRibbonStartRequest true "Tracking Number"
// @Success 200 {object} utils.SuccessResponse{data=models.QCRibbonResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/ribbons/qc-ribbons/start [post]
func (qcrc *QCRibbonController) QCRibbonStart(c fiber.Ctx) error {
	log.Println("QCRibbonStart called")

	// Binding request body
	var req QCRibbonStartRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("QCRibbonStart - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Getting current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Convert tracking number to uppercase and trim spaces
	req.TrackingNumber = strings.ToUpper(strings.TrimSpace(req.TrackingNumber))

	// Check for existing QCRibbon record
	var existingQCRibbon models.QCRibbon
	err = qcrc.DB.Where("tracking_number = ?", req.TrackingNumber).First(&existingQCRibbon).Error
	if err == nil {
		// Record exists
		if existingQCRibbon.Status == "pending" {
			// If status is pending, update it to in_progress
			tx := qcrc.DB.Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()

			existingQCRibbon.Status = "in_progress"
			if err := tx.Save(&existingQCRibbon).Error; err != nil {
				tx.Rollback()
				log.Println("QCRibbonStart - Failed to update QC Ribbon status:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal memperbarui status QC Ribbon",
				})
			}

			// Update order processing status to qc_progress
			if err := tx.Model(&models.Order{}).Where("tracking_number = ?", req.TrackingNumber).Update("processing_status", "qc_progress").Error; err != nil {
				tx.Rollback()
				log.Println("QCRibbonStart - Failed to update order processing status:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal memperbarui status pemrosesan pesanan",
				})
			}

			if err := tx.Commit().Error; err != nil {
				log.Println("QCRibbonStart - Failed to commit transaction:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal memperbarui proses QC Ribbon",
				})
			}

			// Reload with relationships
			if err := qcrc.DB.Preload("QCRibbonDetails.Box").Preload("QCUser").Where("id = ?", existingQCRibbon.ID).First(&existingQCRibbon).Error; err != nil {
				log.Println("QCRibbonStart - Failed to reload QC Ribbon record:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal mengambil data QC Ribbon",
				})
			}

			log.Println("QCRibbonStart completed successfully (resumed from pending)")
			return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
				Success: true,
				Message: "Proses QC Ribbon berhasil dilanjutkan dari pending",
				Data:    existingQCRibbon.ToResponse(),
			})
		} else if existingQCRibbon.Status != "completed" {
			// If status is not completed and not pending, return error
			log.Println("QCRibbonStart - Tracking number already in QC Ribbon records:", req.TrackingNumber)
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Nomor pelacakan " + req.TrackingNumber + " sudah berada dalam proses QC Ribbon.",
			})
		}
		// If status is completed, we'll continue to create a new record
	} else if err != gorm.ErrRecordNotFound {
		// Database error
		log.Println("QCRibbonStart - Database error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memeriksa data QC Ribbon",
		})
	}

	// Check if tracking number exists in orders and have processing status "picking_completed or qc_pending"
	var order models.Order
	if err := qcrc.DB.Where("tracking_number = ? AND processing_status IN ?", req.TrackingNumber, []string{"picking_completed", "qc_pending"}).First(&order).Error; err != nil {
		log.Println("QCRibbonStart - No order found with tracking number in picking completed status:", req.TrackingNumber)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tidak ditemukan pesanan dengan nomor pelacakan " + req.TrackingNumber + " dalam status picking completed.",
		})
	}

	// Start database transaction
	tx := qcrc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create QCRibbon record and update order processing status to "qc_progress"
	qcRibbon := models.QCRibbon{
		TrackingNumber: req.TrackingNumber,
		QCBy:           uint(userID),
		Status:         "in_progress",
		Complained:     false,
	}

	if err := tx.Create(&qcRibbon).Error; err != nil {
		tx.Rollback()
		log.Println("QCRibbonStart - Failed to create QC Ribbon record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memulai proses QC Ribbon",
		})
	}

	order.ProcessingStatus = "qc_progress"
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		log.Println("QCRibbonStart - Failed to update order processing status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memulai proses QC Ribbon",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Println("QCRibbonStart - Failed to commit transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memulai proses QC Ribbon",
		})
	}

	// Reload the created record with all relationships for response
	if err := qcrc.DB.Preload("QCRibbonDetails.Box").Preload("QCUser").Where("id = ?", qcRibbon.ID).First(&qcRibbon).Error; err != nil {
		log.Println("QCRibbonStart - Failed to reload QC Ribbon record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data QC Ribbon yang dimulai",
		})
	}

	log.Println("QCRibbonStart completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Proses QC Ribbon berhasil dimulai",
		Data:    qcRibbon.ToResponse(),
	})
}

// ValidateQCRibbonProduct validates the QC Ribbon Details items or product by SKU and quantity
// @Summary Validate QC Ribbon Product
// @Description Validate the QC Ribbon Details items or product by SKU and quantity
// @Tags Ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Ribbon ID"
// @Param qcRibbon body ValidateQCRibbonProductRequest true "QC Ribbon Details Items"
// @Success 200 {object} utils.SuccessResponse(data=models.QCRibbonResponse)
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/ribbons/qc-ribbons/{id}/validate [put]
func (qcrc *QCRibbonController) ValidateQCRibbonProduct(c fiber.Ctx) error {
	log.Println("ValidateQCRibbonProduct called")
	// Parse id parameter
	id := c.Params("id")
	var qcRibbon models.QCRibbon
	if err := qcrc.DB.Preload("QCRibbonDetails.Box").Preload("QCUser").Where("id = ?", id).First(&qcRibbon).Error; err != nil {
		log.Println("ValidateQCRibbonProduct - QC Ribbon not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Ribbon dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req ValidateQCRibbonProductRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("ValidateQCRibbonProduct - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak sesuai",
		})
	}

	// Check if QC Ribbon is in progress or pending
	if qcRibbon.Status != "in_progress" && qcRibbon.Status != "pending" {
		log.Println("ValidateQCRibbonProduct - QC Ribbon is not in progress or pending:", qcRibbon.Status)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Ribbon tidak sedang dalam status in progress atau pending",
		})
	}

	// If QC Ribbon is pending, check if the user is the one who marked it as pending
	if qcRibbon.Status == "pending" {
		userIDStr := c.Locals("userId").(string)
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			log.Println("ValidateQCRibbonProduct - Invalid user ID:", err)
			return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "ID pengguna tidak valid",
			})
		}

		if qcRibbon.QCBy != uint(userID) {
			log.Println("ValidateQCRibbonProduct - User is not the one who marked QC Ribbon as pending:", userID)
		}
	}

	// Search the target order by tracking number from QC ribbon record
	var order models.Order
	if err := qcrc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("tracking_number = ?", qcRibbon.TrackingNumber).First(&order).Error; err != nil {
		log.Println("ValidateQCRibbonProduct - No order found with tracking number:", qcRibbon.TrackingNumber)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tidak ditemukan pesanan dengan nomor pelacakan yang diberikan " + qcRibbon.TrackingNumber,
		})
	}

	// Find matching order detail by SKU
	var matchedDetail *models.OrderDetail
	for i := range order.OrderDetails {
		if order.OrderDetails[i].SKU == req.SKU {
			matchedDetail = &order.OrderDetails[i]
			break
		}
	}

	// Check if product SKU exists in order details
	if matchedDetail == nil {
		log.Println("ValidateQCRibbonProduct - Product not found in order details:", req.SKU)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Product dengan SKU " + req.SKU + " tidak ditemukan dalam detail pesanan",
		})
	}

	// Check if quantity matches
	if matchedDetail.Quantity != req.Quantity {
		log.Println("ValidateQCRibbonProduct - Quantity mismatch for product:", req.SKU)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Jumlah untuk SKU %s. Tidak sesuai: %d, Diterima: %d", req.SKU, matchedDetail.Quantity, req.Quantity),
		})
	}

	// Update the is_valid flag to true
	if err := qcrc.DB.Model(&models.OrderDetail{}).Where("id = ?", matchedDetail.ID).Update("is_valid", true).Error; err != nil {
		log.Println("ValidateQCRibbonProduct - Failed to update order detail:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui detail pesanan untuk produk dengan SKU " + req.SKU,
		})
	}

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Item QC Ribbon dengan SKU " + req.SKU + " berhasil divalidasi",
		Data:    qcRibbon.ToResponse(),
	})
}

// CompleteQcRibbon adding box details and marking QC Ribbon as completed
// @Summary Complete QC Ribbon
// @Description Add box details and mark QC Ribbon as completed
// @Tags Ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Ribbon ID"
// @Param qcRibbon body CreateQCRibbonDetailRequest true "QC Ribbon Details"
// @Success 200 {object} utils.SuccessResponse{data=models.QCRibbonResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/ribbons/qc-ribbons/{id}/complete [put]
func (qcrc *QCRibbonController) CompleteQcRibbon(c fiber.Ctx) error {
	log.Println("CompleteQcRibbon called")

	// Parse id parameter
	id := c.Params("id")
	var qcRibbon models.QCRibbon
	if err := qcrc.DB.Preload("QCRibbonDetails.Box").Preload("QCUser").Where("id = ?", id).First(&qcRibbon).Error; err != nil {
		log.Println("CompleteQcRibbon - QC Ribbon not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Ribbon dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req CreateQCRibbonDetailRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("CompleteQcRibbon - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check if QC Ribbon is in progress or pending
	if qcRibbon.Status != "in_progress" && qcRibbon.Status != "pending" {
		log.Println("CompleteQcRibbon - QC Ribbon is not in progress or pending:", qcRibbon.Status)
		return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
			Success: true,
			Message: "QC Ribbon tidak sedang dalam status in proggress atau pending",
			Data:    qcRibbon.ToResponse(),
		})
	}

	// If QC Ribbon is pending, check if the user is the one who marked it as pending
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("CompleteQcRibbon - Invalid user ID:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	if qcRibbon.Status == "pending" && qcRibbon.QCBy != uint(userID) {
		log.Println("CompleteQcRibbon - User is not the one who marked QC Ribbon as pending:", userID)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Hanya pengguna yang menandai QC Ribbon sebagai pending yang dapat menyelesaikan",
		})
	}

	// Check if order details have been validated
	var order models.Order
	if err := qcrc.DB.Preload("OrderDetails").Where("tracking_number = ?", qcRibbon.TrackingNumber).First(&order).Error; err != nil {
		log.Println("CompleteQcRibbon - No order found with tracking number:", qcRibbon.TrackingNumber)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tidak ditemukan pesanan dengan nomor pelacakan yang diberikan " + qcRibbon.TrackingNumber,
		})
	}

	for _, detail := range order.OrderDetails {
		if !detail.IsValid {
			log.Println("CompleteQcRibbon - Order details not validated:", qcRibbon.TrackingNumber)
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Detail pesanan belum divalidasi",
			})
		}
	}

	// Validate all boxes exist and no duplicates
	boxIDSet := make(map[uint]bool)
	for _, detailReq := range req.Details {
		// Check for duplicate box IDs in the request
		if boxIDSet[detailReq.BoxID] {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Terdapat ID box duplikat dalam permintaan",
			})
		}
		boxIDSet[detailReq.BoxID] = true
	}

	// Start database transaction
	tx := qcrc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create QCRibbonDetails records
	for _, detailReq := range req.Details {
		qcRibbonDetail := models.QCRibbonDetail{
			QCRibbonID: qcRibbon.ID,
			BoxID:      detailReq.BoxID,
			Quantity:   detailReq.Quantity,
		}
		qcRibbon.QCRibbonDetails = append(qcRibbon.QCRibbonDetails, qcRibbonDetail)
	}
	if err := tx.Create(&qcRibbon.QCRibbonDetails).Error; err != nil {
		tx.Rollback()
		log.Println("CompleteQcRibbon - Failed to create QC Ribbon details:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat detail QC Ribbon",
		})
	}

	// Update QC Ribbon status to completed
	qcRibbon.Status = "completed"
	if err := tx.Save(&qcRibbon).Error; err != nil {
		tx.Rollback()
		log.Println("CompleteQcRibbon - Failed to update QC Ribbon status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyelesaikan QC Ribbon",
		})
	}

	// Update order processing status to "qc_completed"
	if err := tx.Model(&models.Order{}).Where("tracking_number = ?", qcRibbon.TrackingNumber).Update("processing_status", "qc_completed").Error; err != nil {
		tx.Rollback()
		log.Println("CompleteQcRibbon - Failed to update order processing status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pemrosesan pessanan",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Println("CompleteQcRibbon - Failed to commit transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal melakukan commit transaksi",
		})
	}

	// Reload the updated record with all relationships for response
	if err := qcrc.DB.Preload("QCRibbonDetails.Box").Preload("QCUser").First(&qcRibbon, qcRibbon.ID).Error; err != nil {
		log.Println("CompleteQcRibbon - Failed to load updated QC Ribbon:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat QC Ribbon yang telah diperbarui",
		})
	}

	// load order by tracking number
	if err := qcrc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("tracking_number = ?", qcRibbon.TrackingNumber).First(&order).Error; err == nil {
		qcRibbon.Order = &order
	}

	log.Println("CompleteQcRibbon completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "QC Ribbon berhasil diselesaikan",
		Data:    qcRibbon.ToResponse(),
	})
}

// PendingQCRibbon marks a QC Ribbon as pending
// @Summary Pending QC Ribbon
// @Description Mark a QC Ribbon as pending
// @Tags Ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Ribbon ID"
// @Success 200 {object} utils.SuccessResponse(data=models.QCRibbonResponse)
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/ribbons/qc-ribbons/{id}/pending [put]
func (qcrc *QCRibbonController) PendingQCRibbon(c fiber.Ctx) error {
	log.Println("PendingQCRibbon called")

	// Parse id parameter
	id := c.Params("id")
	var qcRibbon models.QCRibbon
	if err := qcrc.DB.Preload("QCRibbonDetails.Box").Preload("QCUser").Where("id = ?", id).First(&qcRibbon).Error; err != nil {
		log.Println("PendingQCRibbon - QC Ribbon not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Ribbon dengan id " + id + " tidak ditemukan.",
		})
	}

	// Check if QC Ribbon is in progress
	if qcRibbon.Status != "in_progress" {
		log.Println("PendingQCRibbon - QC Ribbon is not in progress:", qcRibbon.Status)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Ribbon tidak sedang dalam status in progress",
		})
	}

	// Update QC Ribbon status to pending
	qcRibbon.Status = "pending"
	if err := qcrc.DB.Save(&qcRibbon).Error; err != nil {
		log.Println("PendingQCRibbon - Failed to update QC Ribbon status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menandai QC Ribbon sebagai pending",
		})
	}

	// Update order processing status to "qc_pending"
	if err := qcrc.DB.Model(&models.Order{}).Where("tracking_number = ?", qcRibbon.TrackingNumber).Update("processing_status", "qc_pending").Error; err != nil {
		log.Println("PendingQCRibbon - Failed to update order processing status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pemrosesan pesanan",
		})
	}

	// Reload the updated record with all relationships for response
	if err := qcrc.DB.Preload("QCRibbonDetails.Box").Preload("QCUser").First(&qcRibbon, qcRibbon.ID).Error; err != nil {
		log.Println("PendingQCRibbon - Failed to load updated QC Ribbon:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat QC Ribbon yang telah diperbarui",
		})
	}

	// Load order by tracking number
	var order models.Order
	if err := qcrc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("tracking_number = ?", qcRibbon.TrackingNumber).First(&order).Error; err == nil {
		qcRibbon.Order = &order
	}

	log.Println("PendingQCRibbon completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "QC Ribbon berhasil ditandai sebagai pending",
		Data:    qcRibbon.ToResponse(),
	})
}
