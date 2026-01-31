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

type PickedOrderController struct {
	DB *gorm.DB
}

func NewPickedOrderController(db *gorm.DB) *PickedOrderController {
	return &PickedOrderController{DB: db}
}

// GetPickedOrders retrieves a list of picked orders with pagination and search
// @Summary Get Picked Orders
// @Description Retrieve a list of picked orders with pagination and search
// @Tags Picked Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of picked orders per page" default(10)
// @Param startDate query string false "Start date (YYYY-MM-DD format)"
// @Param endDate query string false "End date (YYYY-MM-DD format)"
// @Param userId query string false "filter term for user ID"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.PickedOrderResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/picked-orders [get]
func (poc *PickedOrderController) GetPickedOrders(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var pickedOrders []models.PickedOrder

	// Build base query
	query := poc.DB.Model(&models.PickedOrder{}).Preload("PickUser").Preload("Order").Preload("Order.OrderDetails").Preload("Order.AssignUser").Preload("Order.PickUser").Preload("Order.PendingUser").Preload("Order.ChangeUser").Preload("Order.DuplicateUser").Preload("Order.CancelUser").Order("created_at DESC")

	// Date range filtering
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	if startDate != "" {
		// Parse start date and set time to beginning of the day
		parsedStartDate, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			log.Println("Invalid start_date format:", err)
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format start_date tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
		startOfDay := time.Date(parsedStartDate.Year(), parsedStartDate.Month(), parsedStartDate.Day(), 0, 0, 0, 0, parsedStartDate.Location())
		query = query.Where("created_at >= ?", startOfDay)
	}
	if endDate != "" {
		// Parse end date and set time to end of the day
		parsedEndDate, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			log.Println("Invalid end_date format:", err)
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format end_date tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
		endOfDay := time.Date(parsedEndDate.Year(), parsedEndDate.Month(), parsedEndDate.Day(), 23, 59, 59, 0, parsedEndDate.Location())
		query = query.Where("created_at <= ?", endOfDay)
	}

	// Filter by user ID if provided exact match
	userId := c.Query("userId")
	if userId != "" {
		query = query.Where("picked_by = ?", userId)
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Offset(offset).Limit(limit).Find(&pickedOrders).Error; err != nil {
		log.Println("Error retrieving picked orders:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data pesanan yang sudah dipick.",
		})
	}

	// Format response
	pickedOrderList := make([]models.PickedOrderResponse, len(pickedOrders))
	for i, pickedOrder := range pickedOrders {
		pickedOrderList[i] = *pickedOrder.ToResponse()
	}

	// Build success message
	message := "Picked orders retrieved successfully."
	var filters []string

	if startDate != "" || endDate != "" {
		var dateRange []string
		if startDate != "" {
			dateRange = append(dateRange, "from: "+startDate)
		}
		if endDate != "" {
			dateRange = append(dateRange, "to: "+endDate)
		}
		filters = append(filters, "date: "+strings.Join(dateRange, ", "))
	}

	if userId != "" {
		filters = append(filters, "userId: "+userId)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return paginated response
	log.Println(message)
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    pickedOrderList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetPickOrder retrieves a single picked order by ID
// @Summary Get Picked Order
// @Description Retrieve a single picked order by ID
// @Tags Picked Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Picked Order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.PickedOrderResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/picked-orders/{id} [get]
func (poc *PickedOrderController) GetPickedOrder(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var pickedOrder models.PickedOrder
	if err := poc.DB.Preload("PickUser").Preload("Order").Preload("Order.OrderDetails").Preload("Order.AssignUser").Preload("Order.PickUser").Preload("Order.PendingUser").Preload("Order.ChangeUser").Preload("Order.DuplicateUser").Preload("Order.CancelUser").First(&pickedOrder, id).Error; err != nil {
		log.Println("Picked order with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan yang dipick dengan id " + id + " tidak ditemukan.",
		})
	}

	log.Println("Picked order retrieved successfully.")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan yang sudah dipick berhasil diambil.",
		Data:    pickedOrder.ToResponse(),
	})
}
