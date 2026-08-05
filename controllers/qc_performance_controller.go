package controllers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"livo-fiber-backend/models"
	"livo-fiber-backend/utils"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type QCPerformanceController struct {
	DB *gorm.DB
}

func NewQCPerformanceController(db *gorm.DB) *QCPerformanceController {
	return &QCPerformanceController{DB: db}
}

// GetQCPerformances gets a list of QC performances with filters and pagination
// @Summary Get QC Performances
// @Description Retrieve a list of QC Performances with pagination, filters, and search
// @Tags Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of QC Performances per page" default(10)
// @Param search query string false "Search term for user name or username"
// @Param user_id query int false "Filter by User ID"
// @Param role query string false "Filter by role (e.g. qc-online, qc-ribbon)"
// @Param start_date query string false "Start Date filter (YYYY-MM-DD)"
// @Param end_date query string false "End Date filter (YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.QCPerformanceResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/reports/qc-performances [get]
func (qc *QCPerformanceController) GetQCPerformances(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var performances []models.QCPerformance

	// Build base query with eager loading
	query := qc.DB.Model(&models.QCPerformance{}).Preload("User").Preload("Details").Where("qc_performances.total_qc > ?", 0).Order("login_time DESC")

	var filters []string

	// Filter by user ID
	userID := c.Query("user_id")
	if userID != "" {
		query = query.Where("qc_performances.user_id = ?", userID)
		filters = append(filters, "user_id: "+userID)
	}

	// Filter by role
	role := c.Query("role")
	if role != "" {
		query = query.Where("qc_performances.role = ?", role)
		filters = append(filters, "role: "+role)
	}

	// Filter by date range (on login_time)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	if startDate != "" && endDate != "" {
		query = query.Where("qc_performances.login_time BETWEEN ? AND ?", startDate, endDate+" 23:59:59")
		filters = append(filters, fmt.Sprintf("date: %s to %s", startDate, endDate))
	} else if startDate != "" {
		query = query.Where("qc_performances.login_time >= ?", startDate)
		filters = append(filters, "start_date: "+startDate)
	} else if endDate != "" {
		query = query.Where("qc_performances.login_time <= ?", endDate+" 23:59:59")
		filters = append(filters, "end_date: "+endDate)
	}

	// Search condition (by user's name or username via join)
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Joins("JOIN users ON users.id = qc_performances.user_id").
			Where("users.name ILIKE ? OR users.username ILIKE ?", "%"+search+"%", "%"+search+"%")
		filters = append(filters, "search: "+search)
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Limit(limit).Offset(offset).Find(&performances).Error; err != nil {
		log.Println("Error retrieving qc performances:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data performa QC",
		})
	}

	// Format response
	performanceList := make([]models.QCPerformanceResponse, len(performances))
	for i, perf := range performances {
		performanceList[i] = *perf.ToResponse()
	}

	// Build success message
	message := "QC performances retrieved successfully"
	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    performanceList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetQCPerformance gets a single QC performance by ID
// @Summary Get QC Performance by ID
// @Description Retrieve a single QC Performance by its ID
// @Tags Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QC Performance ID"
// @Success 200 {object} utils.SuccessResponse{data=models.QCPerformanceResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/reports/qc-performances/{id} [get]
func (qc *QCPerformanceController) GetQCPerformance(c fiber.Ctx) error {
	id := c.Params("id")

	var performance models.QCPerformance
	if err := qc.DB.Preload("User").Preload("Details").First(&performance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Data performa QC tidak ditemukan",
			})
		}
		log.Println("Error retrieving qc performance:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Terjadi kesalahan saat mengambil data performa QC",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "QC performance retrieved successfully",
		"data":    performance.ToResponse(),
	})
}
