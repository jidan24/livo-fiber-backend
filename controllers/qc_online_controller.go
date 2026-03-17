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

type QCOnlineController struct {
	DB *gorm.DB
}

func NewQCOnlineController(db *gorm.DB) *QCOnlineController {
	return &QCOnlineController{DB: db}
}

// Request structs
type QCOnlineStartRequest struct {
	TrackingNumber string `json:"trackingNumber" validate:"required"`
}

type ValidateQCOnlineProductRequest struct {
	SKU      string `json:"sku" validate:"required"`
	Quantity int    `json:"quantity" validate:"required,min=1"`
}

type CreateQCOnlineDetail struct {
	BoxID    uint `json:"boxId" validate:"required"`
	Quantity int  `json:"quantity" validate:"required,min=1"`
}

type CreateQCOnlineDetailRequest struct {
	Details []CreateQCRibbonDetail `json:"details" validate:"required,dive,required"`
}

// Unique response structs
// QcOnlineDailyCount represents the count of qc-onlines for a specific date
type QcOnlineDailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// QcOnlinesDailyCountResponse represents the response for daily qc-online counts
type QcOnlinesDailyCountResponse struct {
	Month       string               `json:"month"`
	Year        int                  `json:"year"`
	DailyCounts []QcOnlineDailyCount `json:"dailyCounts"`
	TotalCount  int                  `json:"totalCount"`
}

// GetQCOnlines retrieves a list of qc onlines with pagination and search
// @Summary Get QC Onlines
// @Description Retrieve a list of QC Onlines with pagination and search
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of QC Onlines per page" default(10)
// @Param search query string false "Search term for tracking number"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.QCOnlineResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/qc-onlines [get]
func (qcoc *QCOnlineController) GetQCOnlines(c fiber.Ctx) error {
	log.Println("GetQCOnlines called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var qcOnlines []models.QCOnline

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
	query := qcoc.DB.Model(&models.QCOnline{}).Preload("QCOnlineDetails.Box").Preload("QCUser").Order("updated_at DESC").Where("qc_by = ?", uint(userID)).Where("updated_at >= ? AND updated_at < ?", startOfDay, endOfDay)

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("tracking_number ILIKE ?", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&qcOnlines).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data QC Online",
		})
	}

	// Load orders for each QC Online by tracking number
	for i := range qcOnlines {
		var order models.Order
		if err := qcoc.DB.Preload("OrderDetails").Where("tracking_number = ?", qcOnlines[i].TrackingNumber).First(&order).Error; err == nil {
			qcOnlines[i].Order = &order
		}
	}

	// Format response
	qcOnlineList := make([]models.QCOnlineResponse, len(qcOnlines))
	for i, qcOnline := range qcOnlines {
		qcOnlineList[i] = *qcOnline.ToResponse()
	}

	// Build success message
	message := "QC Onlines retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println("GetQCOnlines completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    qcOnlineList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetQCOnline retrieves a single qc online by ID
// @Summary Get QC Online
// @Description Retrieve a single QC Online by ID
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Online ID"
// @Success 200 {object} utils.SuccessResponse{data=models.QCOnlineResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/qc-onlines/{id} [get]
func (qcoc *QCOnlineController) GetQCOnline(c fiber.Ctx) error {
	log.Println("GetQCOnline called")
	// Parse id parameter
	id := c.Params("id")
	var qcOnline models.QCOnline
	if err := qcoc.DB.Preload("QCOnlineDetails.Box").Preload("QCUser").Where("id = ?", id).First(&qcOnline).Error; err != nil {
		log.Println("GetQCOnline - QC Online not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Online dengan id " + id + " tidak ditemukan.",
		})
	}

	// Load order by tracking number
	var order models.Order
	if err := qcoc.DB.Preload("OrderDetails").Where("tracking_number = ?", qcOnline.TrackingNumber).First(&order).Error; err == nil {
		qcOnline.Order = &order
	}

	log.Println("GetQCOnline completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "QC Online berhasil diambil",
		Data:    qcOnline.ToResponse(),
	})
}

// GetChartQcOnlines retrieves QC Online data for charting
// @Summary Get Chart QC Onlines
// @Description Retrieve QC Online data for charting
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.SuccessResponse{data=QcOnlinesDailyCountResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/qc-onlines/chart [get]
func (qcoc *QCOnlineController) GetChartQCOnlines(c fiber.Ctx) error {
	log.Println("GetChartQCOnlines called")
	// Get current month start and end dates
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	// Start of the month
	startOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

	// First day of next month at 00:00:00 (to use as upper bound)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	// Query to get daily counts for current month
	var dailyCounts []QcOnlineDailyCount

	if err := qcoc.DB.Model(&models.QCOnline{}).Select("DATE(created_at) as date, COUNT(*) as count").Where("created_at >= ? AND created_at < ?", startOfMonth, startOfNextMonth).Group("DATE(created_at)").Order("date ASC").Scan(&dailyCounts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data QC Online",
		})
	}

	// Get total count for the current month
	var totalCount int64
	if err := qcoc.DB.Model(&models.QCOnline{}).Where("created_at >= ? AND created_at < ?", startOfMonth, startOfNextMonth).Count(&totalCount).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil total jumlah QC Online",
		})
	}

	// Format response
	response := QcOnlinesDailyCountResponse{
		Month:       currentMonth.String(),
		Year:        currentYear,
		DailyCounts: dailyCounts,
		TotalCount:  int(totalCount),
	}

	message := "QC Online chart data " + currentMonth.String() + "  " + strconv.Itoa(currentYear) + " retrieved successfully"

	log.Println("GetChartQCOnlines completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: message,
		Data:    response,
	})
}

// QCOnlineStart Starting QC Online processing for an order
// @Summary Start QC Online Processing
// @Description Mark an order as in QC Online processing
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param trackingNumber body QCRibbonStartRequest true "Tracking Number"
// @Success 200 {object} utils.SuccessResponse{data=models.QCOnlineResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/qc-onlines/start [post]
func (qcoc *QCOnlineController) QCOnlineStart(c fiber.Ctx) error {
	log.Println("QCOnlineStart called")

	// Binding request body
	var req QCOnlineStartRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("QCOnlineStart - Invalid request body:", err)
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

	// Check for existing QCOnline record
	var existingQCOnline models.QCOnline
	err = qcoc.DB.Where("tracking_number = ?", req.TrackingNumber).First(&existingQCOnline).Error
	if err == nil {
		// Record exists
		if existingQCOnline.Status == "pending" {
			// If status is pending, update it to in_progress
			tx := qcoc.DB.Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()

			existingQCOnline.Status = "in_progress"
			if err := tx.Save(&existingQCOnline).Error; err != nil {
				tx.Rollback()
				log.Println("QCOnlineStart - Failed to update QC Online status:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal memperbarui status QC Online",
				})
			}

			// Update order processing status to qc_progress
			if err := tx.Model(&models.Order{}).Where("tracking_number = ?", req.TrackingNumber).Update("processing_status", "qc_progress").Error; err != nil {
				tx.Rollback()
				log.Println("QCOnlineStart - Failed to update order processing status:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal memperbarui status pemrosesan pesanan",
				})
			}

			if err := tx.Commit().Error; err != nil {
				log.Println("QCOnlineStart - Failed to commit transaction:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal memperbarui proses QC Online",
				})
			}

			// Reload with relationships
			if err := qcoc.DB.Preload("QCOnlineDetails.Box").Preload("QCUser").Where("id = ?", existingQCOnline.ID).First(&existingQCOnline).Error; err != nil {
				log.Println("QCOnlineStart - Failed to reload QC Online record:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal mengambil data QC Online",
				})
			}

			log.Println("QCOnlineStart completed successfully (resumed from pending)")
			return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
				Success: true,
				Message: "Proses QC Online berhasil dilanjutkan dari pending",
				Data:    existingQCOnline.ToResponse(),
			})
		} else if existingQCOnline.Status != "completed" {
			// If status is not completed and not pending, return error
			log.Println("QCOnlineStart - Tracking number already in QC Online records:", req.TrackingNumber)
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Nomor pelacakan " + req.TrackingNumber + " sudah berada dalam proses QC Online.",
			})
		}
		// If status is completed, we'll continue to create a new record
	} else if err != gorm.ErrRecordNotFound {
		// Database error
		log.Println("QCOnlineStart - Database error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memeriksa data QC Online",
		})
	}

	// Check if tracking number exists in orders and have processing status "picking_completed or qc_pending"
	var order models.Order
	if err := qcoc.DB.Where("tracking_number = ? AND processing_status IN ?", req.TrackingNumber, []string{"picking_completed", "qc_pending"}).First(&order).Error; err != nil {
		log.Println("QCOnlineStart - No order found with tracking number in picking completed status:", req.TrackingNumber)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tidak ditemukan pesanan dengan nomor pelacakan " + req.TrackingNumber + " dalam status picking completed.",
		})
	}

	// Start database transaction
	tx := qcoc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create QCOnline record and update order processing status to "qc_progress"
	qcOnline := models.QCOnline{
		TrackingNumber: req.TrackingNumber,
		QCBy:           uint(userID),
		Status:         "in_progress",
		Complained:     false,
	}

	if err := tx.Create(&qcOnline).Error; err != nil {
		tx.Rollback()
		log.Println("QCOnlineStart - Failed to create QC Online record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memulai proses QC Online",
		})
	}

	order.ProcessingStatus = "qc_progress"
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		log.Println("QCOnlineStart - Failed to update order processing status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pemrosesan pesanan",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Println("QCOnlineStart - Failed to commit transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memulai proses QC Online",
		})
	}

	// Reload the created record with all relationships for response
	if err := qcoc.DB.Preload("QCOnlineDetails.Box").Preload("QCUser").Where("id = ?", qcOnline.ID).First(&qcOnline).Error; err != nil {
		log.Println("QCOnlineStart - Failed to reload QC Online record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data QC Online yang dimulai",
		})
	}

	log.Println("QCOnlineStart completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Proses QC Online berhasil dimulai",
		Data:    qcOnline.ToResponse(),
	})
}

// ValidateQCOnlineProduct validates the QC Online Details items or product by SKU and quantity
// @Summary Validate QC Online Product
// @Description Validate the QC Online Details items or product by SKU and quantity
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Online ID"
// @Param qcOnline body ValidateQCOnlineProductRequest true "QC Online Details Items"
// @Success 200 {object} utils.SuccessResponse{data=models.QCOnlineDetailResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/qc-onlines/{id}/validate [post]
func (qcoc *QCOnlineController) ValidateQCOnlineProduct(c fiber.Ctx) error {
	log.Println("ValidateQCOnlineProduct called")
	// Parse id parameter
	id := c.Params("id")
	var qcOnline models.QCOnline
	if err := qcoc.DB.Preload("QCOnlineDetails.Box").Where("id = ?", id).First(&qcOnline).Error; err != nil {
		log.Println("ValidateQCOnlineProduct - QC Online not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Online dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req ValidateQCOnlineProductRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("ValidateQCOnlineProduct - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check if QC Online is in progress or pending
	if qcOnline.Status != "in_progress" && qcOnline.Status != "pending" {
		log.Println("ValidateQCOnlineProduct - QC Online is not in progress or pending:", qcOnline.Status)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Online tidak sedang dalam status in progress atau pending",
		})
	}

	// If QC Online is pending, check if the user is the one who marked it as pending
	if qcOnline.Status == "pending" {
		userIDStr := c.Locals("userId").(string)
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "ID pengguna tidak valid",
			})
		}

		if qcOnline.QCBy != uint(userID) {
			log.Println("ValidateQCOnlineProduct - User is not the one who marked QC Online as pending:", userID)
		}
	}

	// Search the target order by tracking number from QC online record
	var order models.Order
	if err := qcoc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("tracking_number = ?", qcOnline.TrackingNumber).First(&order).Error; err != nil {
		log.Println("ValidateQCOnlineProduct - No order found with tracking number:", qcOnline.TrackingNumber)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tidak ditemukan pesanan dangan nomor pelacakan yang diberikan " + qcOnline.TrackingNumber,
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
		log.Println("ValidatedQCOnlineProduct - Product not found in order details:", req.SKU)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Product dengan SKU " + req.SKU + " tidak ditemukan detail pesanan.",
		})
	}

	// Check if quantity matches
	if matchedDetail.Quantity != req.Quantity {
		log.Println("ValidateQCOnlineProduct - Quantity mismatch for product:", req.SKU)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Jumlah untuk SKU tidak sesuai %s. Diharapkan: %d, Diterima: %d", req.SKU, matchedDetail.Quantity, req.Quantity),
		})
	}

	// Update the is_valid flag to true
	if err := qcoc.DB.Model(&models.OrderDetail{}).Where("id = ?", matchedDetail.ID).Update("is_valid", true).Error; err != nil {
		log.Println("ValidateQCOnlineProduct - Failed to update order detail:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui detail pesanan untuk produk dengan SKU " + req.SKU,
		})
	}

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Produk dengan SKU " + req.SKU + " berhasil divalidasi.",
		Data:    qcOnline.ToResponse(),
	})
}

// CompleteQcOnline adding box details and marking QC Online as completed.
// @Summary Complete QC Online
// @Description Add box details and mark QC Online as completed
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Online ID"
// @Param qcOnline body CreateQCOnlineDetailRequest true "QC Online Box Details"
// @Success 200 {object} utils.SuccessResponse{data=models.QCOnlineResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/qc-onlines/{id}/complete [put]
func (qcoc *QCOnlineController) CompleteQcOnline(c fiber.Ctx) error {
	log.Println("CompleteQcOnline called")

	// Parse id parameter
	id := c.Params("id")
	var qcOnline models.QCOnline
	if err := qcoc.DB.Preload("QCOnlineDetails.Box").Where("id = ?", id).First(&qcOnline).Error; err != nil {
		log.Println("CompleteQcOnline - QC Online not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Online dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req CreateQCOnlineDetailRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("CompleteQcOnline - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check if QC Online is in progress or pending
	if qcOnline.Status != "in_progress" && qcOnline.Status != "pending" {
		log.Println("CompleteQcOnline - QC Online is not in progress or pending:", qcOnline.Status)
		return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
			Success: true,
			Message: "QC Online tidak sedang dalam status in progress atau pending",
			Data:    qcOnline.ToResponse(),
		})
	}

	// If QC Online is pending, check if the user is the one who marked it as pending
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("CompleteQcOnline - Invalid user ID:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	if qcOnline.Status == "pending" && qcOnline.QCBy != uint(userID) {
		log.Println("CompleteQcOnline - User is not the one who marked QC Online as pending:", userID)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Hanya pengguna yang menandai QC Online sebagai pending yang dapat menyelesaikannya",
		})
	}

	// Check if order details have been validated
	var order models.Order
	if err := qcoc.DB.Preload("OrderDetails").Where("tracking_number = ?", qcOnline.TrackingNumber).First(&order).Error; err != nil {
		log.Println("CompleteQcOnline - No order found with tracking number:", qcOnline.TrackingNumber)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tidak ditemukan pesanan dengan nomor pelacakan yang diberikan " + qcOnline.TrackingNumber,
		})
	}

	for _, detail := range order.OrderDetails {
		if !detail.IsValid {
			log.Println("CompleteQcOnline - Order details not validated:", qcOnline.TrackingNumber)
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
	tx := qcoc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create QCOnlineDetails records
	for _, detailReq := range req.Details {
		qcOnlineDetail := models.QCOnlineDetail{
			QCOnlineID: qcOnline.ID,
			BoxID:      detailReq.BoxID,
			Quantity:   detailReq.Quantity,
		}
		qcOnline.QCOnlineDetails = append(qcOnline.QCOnlineDetails, qcOnlineDetail)
	}
	if err := tx.Create(&qcOnline.QCOnlineDetails).Error; err != nil {
		tx.Rollback()
		log.Println("CompleteQcOnline - Failed to create QC Online details:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat detail QC Online",
		})
	}

	// Update QCOnline status to completed
	qcOnline.Status = "completed"
	if err := tx.Save(&qcOnline).Error; err != nil {
		tx.Rollback()
		log.Println("CompleteQcOnline - Failed to update QC Online status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status QC Online",
		})
	}

	// Update order processing status to "qc_completed"
	if err := tx.Model(&models.Order{}).Where("tracking_number = ?", qcOnline.TrackingNumber).Update("processing_status", "qc_completed").Error; err != nil {
		tx.Rollback()
		log.Println("CompleteQcOnline - Failed to update order processing status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pemrosesan pesanan",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Println("CompleteQcOnline - Failed to commit transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyelesaikan proses QC Online",
		})
	}

	// Reload the updated record with all relationships for response
	if err := qcoc.DB.Preload("QCOnlineDetails.Box").Preload("QCUser").Where("id = ?", qcOnline.ID).First(&qcOnline).Error; err != nil {
		log.Println("CompleteQcOnline - Failed to reload QC Online record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data QC Online yang telah selesai",
		})
	}

	// load order by tracking number
	if err := qcoc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("tracking_number = ?", qcOnline.TrackingNumber).First(&order).Error; err == nil {
		qcOnline.Order = &order
	}

	log.Println("CompleteQcOnline completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Proses QC Online berhasil diselesaikan",
		Data:    qcOnline.ToResponse(),
	})
}

// PendingQCOnline marks a QC Online as pending
// @Summary Pending QC Online
// @Description Mark a QC Online as pending
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Online ID"
// @Success 200 {object} utils.SuccessResponse{data=models.QCOnlineResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/qc-onlines/{id}/pending [put]
func (qcoc *QCOnlineController) PendingQCOnline(c fiber.Ctx) error {
	log.Println("PendingQCOnline called")

	// Parse id parameter
	id := c.Params("id")
	var qcOnline models.QCOnline
	if err := qcoc.DB.Preload("QCOnlineDetails.Box").Preload("QCUser").Where("id = ?", id).First(&qcOnline).Error; err != nil {
		log.Println("PendingQCOnline - QC Online not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Online dengan id " + id + " tidak ditemukan.",
		})
	}

	// Check if QC Online is in progress
	if qcOnline.Status != "in_progress" {
		log.Println("PendingQCOnline - QC Online is not in progress:", qcOnline.Status)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "QC Online tidak sedang dalam status in progress",
		})
	}

	// Update QCOnline status to pending
	qcOnline.Status = "pending"
	if err := qcoc.DB.Save(&qcOnline).Error; err != nil {
		log.Println("PendingQCOnline - Failed to update QC Online status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menandai QC Online sebagai pending",
		})
	}

	// Update order processing status to "qc_pending"
	if err := qcoc.DB.Model(&models.Order{}).Where("tracking_number = ?", qcOnline.TrackingNumber).Update("processing_status", "qc_pending").Error; err != nil {
		log.Println("PendingQCOnline - Failed to update order processing status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pemrosesan pesanan",
		})
	}

	// Reload the updated record with all relationships for response
	if err := qcoc.DB.Preload("QCOnlineDetails.Box").Preload("QCUser").Where("id = ?", qcOnline.ID).First(&qcOnline).Error; err != nil {
		log.Println("PendingQCOnline - Failed to reload QC Online record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data QC Online yang pending",
		})
	}

	// Load order by tracking number
	var order models.Order
	if err := qcoc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("tracking_number = ?", qcOnline.TrackingNumber).First(&order).Error; err == nil {
		qcOnline.Order = &order
	}

	log.Println("PendingQCOnline completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "QC Online berhasil ditandai sebagai pending",
		Data:    qcOnline.ToResponse(),
	})
}
