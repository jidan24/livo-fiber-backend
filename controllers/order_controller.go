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

type OrderController struct {
	DB *gorm.DB
}

func NewOrderController(db *gorm.DB) *OrderController {
	return &OrderController{DB: db}
}

// Request structs
type CreateOrderRequest struct {
	OrderGineeID   string                     `json:"orderGineeId" validate:"required,min=3,max=100"`
	Channel        string                     `json:"channel" validate:"required,min=3,max=100"`
	Store          string                     `json:"store" validate:"required,min=3,max=100"`
	Buyer          string                     `json:"buyer" validate:"required,min=3,max=100"`
	Address        string                     `json:"address" validate:"required,min=3,max=255"`
	Courier        string                     `json:"courier" validate:"omitempty,min=3,max=100"`
	TrackingNumber string                     `json:"trackingNumber" validate:"omitempty,min=3,max=100"`
	SentBefore     string                     `json:"sentBefore" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	Details        []CreateOrderDetailRequest `json:"details" validate:"required,dive,required"`
}

type CreateOrderDetailRequest struct {
	SKU         string `json:"sku" validate:"required,min=1,max=255"`
	ProductName string `json:"productName" validate:"required,min=1,max=255"`
	Variant     string `json:"variant" validate:"omitempty,min=1,max=100"`
	Quantity    int    `json:"quantity" validate:"required,gt=0"`
	Price       int    `json:"price" validate:"required,gt=0"`
}

type BulkCreateOrdersRequest struct {
	Orders []CreateOrderRequest `json:"orders" validate:"required,dive,required"`
}

type UpdateOrderRequest struct {
	Details []UpdateOrderDetailRequest `json:"details" validate:"required,dive,required"`
}

type UpdateOrderDetailRequest struct {
	SKU         string `json:"sku" validate:"required,min=1,max=255"`
	ProductName string `json:"productName" validate:"required,min=1,max=255"`
	Variant     string `json:"variant" validate:"omitempty,min=1,max=100"`
	Quantity    int    `json:"quantity" validate:"required,gt=0"`
	Price       int    `json:"price" validate:"required,gt=0"`
}

type UpdateProcessingStatusRequest struct {
	ProcessingStatus string `json:"processingStatus" validate:"required,min=3,max=50"`
}

type UpdateEventStatusRequest struct {
	EventStatus string `json:"eventStatus" validate:"required,min=3,max=50"`
}

type AssignPickerRequest struct {
	PickerID       uint   `json:"pickerId" validate:"required"`
	TrackingNumber string `json:"trackingNumber" validate:"required,min=3,max=100"`
}

// Unique Response structs
type BulkCreateOrdersReponse struct {
	Summary       BulkCreateSummary      `json:"summary"`
	CreatedOrders []models.OrderResponse `json:"createdOrders"`
	SkippedOrders []SkippedOrder         `json:"skippedOrders"`
	FailedOrders  []FailedOrder          `json:"failedOrders"`
}

type BulkCreateSummary struct {
	Total   uint `json:"total"`
	Created uint `json:"created"`
	Skipped uint `json:"skipped"`
	Failed  uint `json:"failed"`
}

type SkippedOrder struct {
	Index          int    `json:"index"`
	OrderGineeID   string `json:"orderGineeId"`
	TrackingNumber string `json:"trackingNumber"`
	Reason         string `json:"reason"`
}

type FailedOrder struct {
	Index          int    `json:"index"`
	OrderGineeID   string `json:"orderGineeId"`
	TrackingNumber string `json:"trackingNumber"`
	Error          string `json:"error"`
}

type DuplicatedOrderResponse struct {
	OriginalOrder   models.OrderResponse `json:"originalOrder"`
	DuplicatedOrder models.OrderResponse `json:"duplicatedOrder"`
}

// GetOrders retrieves a list of orders with pagination and search
// @Summary Get Orders
// @Description Retrieve a list of orders with pagination and search
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of orders per page" default(10)
// @Param startDate query string false "Start date (YYYY-MM-DD format)"
// @Param endDate query string false "End date (YYYY-MM-DD format)"
// @Param search query string false "Search term for order ginee id or tracking number"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders [get]
func (oc *OrderController) GetOrders(c fiber.Ctx) error {
	log.Println("GetOrders called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var orders []models.Order

	// Build base query
	query := oc.DB.Model(&models.Order{}).Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Order("created_at DESC")

	// Date range filter if provided
	startDate := c.Query("startDate", "")
	endDate := c.Query("endDate", "")
	if startDate != "" {
		// Parse start date and set time to beginning of the day
		parsedStartDate, err := time.Parse("2006-01-02", startDate)
		if err != nil {
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
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format end_date tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
		endOfDay := time.Date(parsedEndDate.Year(), parsedEndDate.Month(), parsedEndDate.Day(), 23, 59, 59, 0, parsedEndDate.Location())
		query = query.Where("created_at <= ?", endOfDay)
	}

	// Search condition if provided
	search := c.Query("search", "")
	if search != "" {
		query = query.Where("order_ginee_id ILIKE ? OR tracking_number ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data pesanan",
		})
	}

	// Load product details in order responses
	for i := range orders {
		for j := range orders[i].OrderDetails {
			var product models.Product
			if err := oc.DB.Where("sku = ?", orders[i].OrderDetails[j].SKU).First(&product).Error; err == nil {
				orders[i].OrderDetails[j].Product = &product
			}
		}
	}

	// Format response
	orderList := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		orderList[i] = *order.ToOrderResponse()
	}

	// Build success message
	message := "Orders retrieved successfully"
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

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println("GetOrders completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    orderList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetOrder retrieves a single order by ID
// @Summary Get Order
// @Description Retrieve a single order by ID
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/{id} [get]
func (oc *OrderController) GetOrder(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := oc.DB.Where("id = ?", id).Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&order).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Order dengan id " + id + " tidak ditemukan.",
		})
	}

	// Load product details in order response
	for i := range order.OrderDetails {
		var product models.Product
		if err := oc.DB.Where("sku = ?", order.OrderDetails[i].SKU).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	log.Println("GetOrder completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil diambil",
		Data:    order.ToOrderResponse(),
	})
}

// CreateOrder creates a new order
// @Summary Create Order
// @Description Create a new order
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param order body CreateOrderRequest true "Order details"
// @Success 201 {object} utils.SuccessResponse{data=models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders [post]
func (oc *OrderController) CreateOrder(c fiber.Ctx) error {
	log.Println("CreateOrder called")
	// Binding request body
	var req CreateOrderRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("CreateOrder - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Convert Order Ginee ID to uppercase and trim spaces
	req.OrderGineeID = strings.ToUpper(strings.TrimSpace(req.OrderGineeID))

	// Convert Tracking Number to uppercase and trim spaces
	req.TrackingNumber = strings.ToUpper(strings.TrimSpace(req.TrackingNumber))

	// Check for existing order with same Order Ginee ID or Tracking Number
	var existingOrder models.Order
	if err := oc.DB.Where("order_ginee_id = ? OR tracking_number = ?", req.OrderGineeID, req.TrackingNumber).First(&existingOrder).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan dengan order ginee id " + req.OrderGineeID + " atau nomor pelacakan " + req.TrackingNumber + " sudah terdaftar.",
		})
	}

	// Parse Sent Before date if provided
	var sentBefore time.Time
	if req.SentBefore != "" {
		var err error
		sentBefore, err = time.Parse("2006-01-02 15:04:00", req.SentBefore)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format sentBefore tidak valid. Gunakan format YYYY-MM-DD HH:MM:SS.",
			})
		}
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Parameter senBefore wajib diisi",
		})
	}

	// Start transaction
	tx := oc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create new order
	newOrder := models.Order{
		OrderGineeID:     req.OrderGineeID,
		ProcessingStatus: "ready_to_pick",
		EventStatus:      "in_progress",
		Channel:          req.Channel,
		Store:            req.Store,
		Buyer:            req.Buyer,
		Address:          req.Address,
		Courier:          req.Courier,
		TrackingNumber:   req.TrackingNumber,
		SentBefore:       sentBefore,
	}

	if err := tx.Create(&newOrder).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat pesanan",
		})
	}

	// Create order details
	for _, detail := range req.Details {
		orderDetail := models.OrderDetail{
			SKU:         detail.SKU,
			ProductName: detail.ProductName,
			Variant:     detail.Variant,
			Quantity:    detail.Quantity,
			Price:       detail.Price,
		}
		newOrder.OrderDetails = append(newOrder.OrderDetails, orderDetail)
	}

	// Save order details within transaction
	if err := tx.Save(&newOrder).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat detail pesanan",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal melakukan commit transaksi",
		})
	}

	// Reload the data
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&newOrder, newOrder.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil dibuat",
		Data:    newOrder.ToOrderResponse(),
	})
}

// BulkCreateOrders creates multiple orders in a single request
// @Summary Bulk Create Orders
// @Description Create multiple orders in a single request
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orders body BulkCreateOrdersRequest true "List of orders to create"
// @Success 201 {object} utils.SuccessResponse{data=BulkCreateOrdersReponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/bulk [post]
func (oc *OrderController) BulkCreateOrders(c fiber.Ctx) error {
	log.Println("BulkCreateOrders called")
	// Binding request body
	var req BulkCreateOrdersRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("BulkCreateOrders - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	var createdOrders []models.Order
	var skippedOrders []SkippedOrder
	var failedOrders []FailedOrder

	for i, orderReq := range req.Orders {
		// Convert Order Ginee ID to uppercase and trim spaces
		orderReq.OrderGineeID = strings.ToUpper(strings.TrimSpace(orderReq.OrderGineeID))

		// Convert Tracking Number to uppercase and trim spaces
		orderReq.TrackingNumber = strings.ToUpper(strings.TrimSpace(orderReq.TrackingNumber))

		// Check if order with same OrderGineeID or tracking number already exists
		var existingOrder models.Order
		if err := oc.DB.Where("order_ginee_id = ? OR tracking_number = ?", orderReq.OrderGineeID, orderReq.TrackingNumber).First(&existingOrder).Error; err == nil {
			// If order already exists, skip it
			skippedOrders = append(skippedOrders, SkippedOrder{
				Index:          i,
				OrderGineeID:   orderReq.OrderGineeID,
				TrackingNumber: orderReq.TrackingNumber,
				Reason:         "Order already exists",
			})
			continue
		}

		// Create order
		order := models.Order{
			OrderGineeID:     orderReq.OrderGineeID,
			ProcessingStatus: "ready_to_pick",
			EventStatus:      "in_progress",
			Channel:          orderReq.Channel,
			Store:            orderReq.Store,
			Buyer:            orderReq.Buyer,
			Address:          orderReq.Address,
			Courier:          orderReq.Courier,
			TrackingNumber:   orderReq.TrackingNumber,
		}

		if orderReq.SentBefore != "" {
			if parsedTime, err := time.Parse("2006-01-02 15:04:00", orderReq.SentBefore); err == nil {
				order.SentBefore = parsedTime
			} else {
				// Failed to parse date
				failedOrders = append(failedOrders, FailedOrder{
					Index:        i,
					OrderGineeID: orderReq.OrderGineeID,
					Error:        "Invalid sentBefore format: " + err.Error(),
				})
				continue
			}
		}

		// Create order details
		for _, detailReq := range orderReq.Details {
			orderDetail := models.OrderDetail{
				SKU:         detailReq.SKU,
				ProductName: detailReq.ProductName,
				Variant:     detailReq.Variant,
				Quantity:    detailReq.Quantity,
				Price:       detailReq.Price,
			}
			order.OrderDetails = append(order.OrderDetails, orderDetail)
		}

		// Try to create the order using transaction
		tx := oc.DB.Begin()
		if err := tx.Create(&order).Error; err != nil {
			tx.Rollback()
			// Failed to create order
			failedOrders = append(failedOrders, FailedOrder{
				Index:        i,
				OrderGineeID: orderReq.OrderGineeID,
				Error:        err.Error(),
			})
			continue
		}
		tx.Commit()

		// Load order with details for response
		oc.DB.Preload("OrderDetails").First(&order, order.ID)
		createdOrders = append(createdOrders, order)
	}

	// Format response
	createdOrderResponses := make([]models.OrderResponse, len(createdOrders))
	for i, order := range createdOrders {
		createdOrderResponses[i] = *order.ToOrderResponse()
	}

	response := BulkCreateOrdersReponse{
		Summary: BulkCreateSummary{
			Total:   uint(len(req.Orders)),
			Created: uint(len(createdOrders)),
			Skipped: uint(len(skippedOrders)),
			Failed:  uint(len(failedOrders)),
		},
		CreatedOrders: createdOrderResponses,
		SkippedOrders: skippedOrders,
		FailedOrders:  failedOrders,
	}

	// Build success message
	statusCode := fiber.StatusCreated
	message := "Bulk order creation completed"

	if len(createdOrders) == 0 {
		if len(skippedOrders) > 0 {
			statusCode = fiber.StatusOK
			message = "All orders were skipped (already exist)"
		} else {
			statusCode = fiber.StatusBadRequest
			message = "No orders could be created"
		}
	} else if len(failedOrders) > 0 || len(skippedOrders) > 0 {
		message = "Bulk order creation completed with some issues"
	}

	// Return response
	log.Printf("BulkCreateOrders completed (created=%d, skipped=%d, failed=%d)\n", len(createdOrders), len(skippedOrders), len(failedOrders))
	return c.Status(statusCode).JSON(utils.SuccessResponse{
		Success: true,
		Message: message,
		Data:    response,
	})
}

// UpdateOrder updates an existing order
// @Summary Update Order
// @Description Update an existing order
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param order body UpdateOrderRequest true "Updated order details"
// @Success 200 {object} utils.SuccessResponse{data=models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/{id} [put]
func (oc *OrderController) UpdateOrder(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("id = ?", id).First(&order).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Order dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdateOrderRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Check if order processing status allows modification
	if order.ProcessingStatus == "picking_progress" || order.ProcessingStatus == "qc_progress" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan tidak dapat diubah dalam status ini " + order.ProcessingStatus + " status.",
		})
	}

	// Check if order is canceled
	if order.EventStatus == "canceled" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan yang dibatalkan tidak dapat diubah.",
		})
	}

	// Start transaction
	tx := oc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update changed_by and changed_at fields
	now := time.Now()
	userIDUint := uint(userID)
	order.ChangedBy = &userIDUint
	order.ChangedAt = &now

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui pesanan",
		})
	}

	// Update order details if provided - replace all details
	if req.Details != nil {
		// Delete all existing order details
		if err := tx.Where("order_id = ?", order.ID).Delete(&models.OrderDetail{}).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Gagal memperbarui detail pesanan",
			})
		}

		// Create new order details
		newDetails := make([]models.OrderDetail, 0, len(req.Details))
		for _, detailReq := range req.Details {
			detail := models.OrderDetail{
				OrderID:     order.ID,
				SKU:         detailReq.SKU,
				ProductName: detailReq.ProductName,
				Variant:     detailReq.Variant,
				Quantity:    detailReq.Quantity,
				Price:       detailReq.Price,
			}
			newDetails = append(newDetails, detail)
		}

		if len(newDetails) > 0 {
			if err := tx.Create(&newDetails).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal memperbarui detail pesanan",
				})
			}
		}

		// Update order's OrderDetails field
		order.OrderDetails = newDetails
	}

	// Coommit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui pesanan",
		})
	}

	// Reload the data with fresh query
	var reloadedOrder models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&reloadedOrder, order.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan",
		})
	}

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil diperbarui",
		Data:    reloadedOrder.ToOrderResponse(),
	})
}

// DuplicateOrder duplicates an existing order
// @Summary Duplicate Order
// @Description Duplicate an existing order
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 201 {object} utils.SuccessResponse{data=DuplicatedOrderResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/{id}/duplicate [put]
func (oc *OrderController) DuplicateOrder(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("id = ?", id).Preload("OrderDetails").First(&order).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Order dengan id " + id + " tidak ditemukan.",
		})
	}

	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Check if order processing status allows modification
	if order.ProcessingStatus == "picking_progress" || order.ProcessingStatus == "qc_progress" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan tidak dapat diduplikasi dalam status ini " + order.ProcessingStatus + " status.",
		})
	}

	// Check if order is canceled
	if order.EventStatus == "canceled" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan yang dibatlkan tidak dapat diduplikasi.",
		})
	}

	// Check if order event status has been duplicated
	if order.EventStatus == "duplicated" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan sudah pernah diduplikasi.",
		})
	}

	// Start transaction
	tx := oc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Store original tracking number before duplication
	originalTrackingNumber := order.TrackingNumber
	newTrackingNumber := "X-" + originalTrackingNumber

	// Update original order's order ginee id by adding "-X2" suffix and tracking number with "X-" prefix
	now := time.Now()
	userIDUint := uint(userID)
	eventStatusDuplicated := "duplicated"
	order.EventStatus = eventStatusDuplicated
	order.OrderGineeID = order.OrderGineeID + "-X2"
	order.TrackingNumber = newTrackingNumber
	order.DuplicatedBy = &userIDUint
	order.DuplicatedAt = &now

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui pesanan asli untuk duplikasi",
		})
	}

	// Update tracking number in qc ribbon, qc online, and outbound if exists (ignore errors if table doesn't exist)
	tx.Model(&models.QCRibbon{}).Where("tracking_number = ?", originalTrackingNumber).Update("tracking_number", newTrackingNumber)
	tx.Model(&models.QCOnline{}).Where("tracking_number = ?", originalTrackingNumber).Update("tracking_number", newTrackingNumber)
	tx.Model(&models.Outbound{}).Where("tracking_number = ?", originalTrackingNumber).Update("tracking_number", newTrackingNumber)

	// Create duplicated order
	duplicatedEventStatus := "duplicated"
	duplicatedOrder := models.Order{
		OrderGineeID:     order.OrderGineeID[:len(order.OrderGineeID)-3], // Remove "-X2" suffix
		ProcessingStatus: order.ProcessingStatus,
		Channel:          order.Channel,
		Store:            order.Store,
		Buyer:            order.Buyer,
		Address:          order.Address,
		Courier:          order.Courier,
		TrackingNumber:   originalTrackingNumber,
		SentBefore:       order.SentBefore,
		EventStatus:      duplicatedEventStatus,
		DuplicatedBy:     &userIDUint,
		DuplicatedAt:     &now,
	}

	// Duplicate order details
	for _, detail := range order.OrderDetails {
		duplicatedDetail := models.OrderDetail{
			SKU:         detail.SKU,
			ProductName: detail.ProductName,
			Variant:     detail.Variant,
			Quantity:    detail.Quantity,
			Price:       detail.Price,
		}
		duplicatedOrder.OrderDetails = append(duplicatedOrder.OrderDetails, duplicatedDetail)
	}

	// Create duplicated order in database
	if err := tx.Create(&duplicatedOrder).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat pesanan duplikasi",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal melakukan commit transaksi",
		})
	}

	// Reload the data with fresh query
	var reloadedOriginalOrder models.Order
	var reloadedDuplicatedOrder models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&reloadedOriginalOrder, order.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan asli",
		})
	}
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&reloadedDuplicatedOrder, duplicatedOrder.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan duplikasi",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil diduplikasi",
		Data: map[string]interface{}{
			"originalOrder":   reloadedOriginalOrder.ToOrderResponse(),
			"duplicatedOrder": reloadedDuplicatedOrder.ToOrderResponse(),
		},
	})
}

// CancelOrder cancels an existing order
// @Summary Cancel Order
// @Description Cancel an existing order
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/{id}/cancel [put]
func (oc *OrderController) CancelOrder(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("id = ?", id).First(&order).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Order dengan id " + id + " tidak ditemukan.",
		})
	}

	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Check if order status allows modification
	if order.ProcessingStatus == "picking_progress" || order.ProcessingStatus == "qc_progress" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Status pesanan tidak memungkinkan pembatalan",
		})
	}

	// Check if order is already cancelled
	if order.EventStatus == "cancelled" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan sudah dibatalkan",
		})
	}

	// Start database transaction
	tx := oc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update order status to cancelled
	now := time.Now()
	userIDUint := uint(userID)
	eventStatusCanceled := "canceled"
	order.EventStatus = eventStatusCanceled
	order.CanceledBy = &userIDUint
	order.CanceledAt = &now

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membatalkan pesanan",
		})
	}

	// Set all order details quantity to zero
	if err := tx.Model(&models.OrderDetail{}).Where("order_id = ?", order.ID).Update("quantity", 0).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui detail pesanan",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal melakukan commit transaksi",
		})
	}

	// Reload the data with fresh query
	var reloadedOrder models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&reloadedOrder, order.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan",
		})
	}

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil dibatalkan",
		Data:    reloadedOrder.ToOrderResponse(),
	})
}

// AssignPicker assigns a picker to an order
// @Summary Assign Picker
// @Description Assign a picker to an order
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param picker body AssignPickerRequest true "Picker assignment details"
// @Success 200 {object} utils.SuccessResponse{data=models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/assign-picker [post]
func (oc *OrderController) AssignPicker(c fiber.Ctx) error {
	log.Println("AssignPicker called")
	// Binding request body
	var req AssignPickerRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("AssignPicker - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Get target order by tracking number
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("tracking_number = ?", req.TrackingNumber).First(&order).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan dengan nomor pelacakan " + req.TrackingNumber + " tidak ditemukan.",
		})
	}

	// Check if the picker is exists
	var picker models.User
	if err := oc.DB.First(&picker, "id = ?", req.PickerID).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Picker dengan id " + strconv.FormatUint(uint64(req.PickerID), 10) + " tidak ada.",
		})
	}

	// Check if order processing status allows assignment
	if order.ProcessingStatus != "ready_to_pick" && order.ProcessingStatus != "picking_pending" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan tidak dapat diberikan picker dalam status ini" + order.ProcessingStatus + " status.",
		})
	}

	// Check if order is canceled
	if order.EventStatus == "canceled" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan yang dibatalkan tidak dapat diberikan picker.",
		})
	}

	// Update order with assignment details
	now := time.Now()
	userIDUint := uint(userID)
	order.AssignedBy = &userIDUint
	order.AssignedAt = &now
	order.PickedBy = &req.PickerID
	order.ProcessingStatus = "picking_progress"

	if err := oc.DB.Save(&order).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memberikan picker pada pesanan",
		})
	}

	// Reload the data with fresh query
	var reloadedOrder models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&reloadedOrder, order.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan",
		})
	}

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Picker berhasil ditugaskan pada pesanan",
		Data:    reloadedOrder.ToOrderResponse(),
	})
}

// PendingPickingOrders marks an order as pending picking
// @Summary Pending Picking Order
// @Description Mark an order as pending picking
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/{id}/pending-picking [put]
func (oc *OrderController) PendingPickingOrders(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("id = ?", id).First(&order).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Order dengan id " + id + " tidak ditemukan.",
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

	// Check if order processing status is "picking_progress"
	if order.ProcessingStatus != "picking_progress" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan tidak dapat ditandai sebagai pending dalam status ini " + order.ProcessingStatus + " status.",
		})
	}

	// Update order to pending picking
	now := time.Now()
	userIDUint := uint(userID)
	order.ProcessingStatus = "picking_pending"
	order.PendingBy = &userIDUint
	order.PendingAt = &now
	order.PickedBy = nil
	order.AssignedBy = nil
	order.AssignedAt = nil

	if err := oc.DB.Select("ProcessingStatus", "PendingBy", "PendingAt", "PickedBy", "AssignedBy", "AssignedAt").Save(&order).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menandai pesanan sebagai pending",
		})
	}

	// Reload the data with fresh query
	var reloadedOrder models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&reloadedOrder, order.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan",
		})
	}

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil ditandai sebagai pending picking",
		Data:    reloadedOrder.ToOrderResponse(),
	})
}

// GetAssignedOrders retrieves orders assigned to a all picker
// @Summary Get Assigned Orders
// @Description Retrieve orders assigned to a all picker with pagination, date range filtering, and search
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param start_date query string false "Start date for filtering (YYYY-MM-DD)"
// @Param end_date query string false "End date for filtering (YYYY-MM-DD)"
// @Param search query string false "Search term for filtering"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/assigned [get]
func (oc *OrderController) GetAssignedOrders(c fiber.Ctx) error {
	log.Println("GetAssignedOrders called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var orders []models.Order

	// Build base query
	query := oc.DB.Model(&models.Order{}).Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Order("created_at DESC").Where("processing_status = ?", "picking_progress")

	// Date range filter if provided
	startDate := c.Query("start_date", "")
	endDate := c.Query("end_date", "")
	if startDate != "" {
		// Parse start date and set time to beginning of the day
		parsedStartDate, err := time.Parse("2006-01-02", startDate)
		if err != nil {
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
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format end_date tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
		endOfDay := time.Date(parsedEndDate.Year(), parsedEndDate.Month(), parsedEndDate.Day(), 23, 59, 59, 0, parsedEndDate.Location())
		query = query.Where("created_at <= ?", endOfDay)
	}

	// Search condition if provided
	search := c.Query("search", "")
	if search != "" {
		query = query.Where("order_ginee_id ILIKE ? OR tracking_number ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data pesanan",
		})
	}

	// Format response
	orderList := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		orderList[i] = *order.ToOrderResponse()
	}

	// Build success message
	message := "Assigned orders retrieved successfully"
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

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return response
	log.Println("GetAssignedOrders completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    orderList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// QcProcessStatusUpdate updates the QC process status of an order
// @Summary Update QC Process Status
// @Description Update the QC process status of an order
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/{id}/status/qc-process [put]
func (oc *OrderController) QCProcessStatusUpdate(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("id = ?", id).First(&order).Error; err != nil {
		log.Println("QCProcessStatusUpdate - Order not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Order dengan id " + id + " tidak ditemukan.",
		})
	}

	// Check if order processing status allows modification
	if order.ProcessingStatus == "qc_progress" {
		log.Println("QCProcessStatusUpdate - Order is already in qc process status.")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan sudah berada dalam status proses qc.",
		})
	}

	// Check if order is canceled
	if order.EventStatus == "canceled" {
		log.Println("QCProcessStatusUpdate - Canceled order cannot be updated to qc process status.")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan yang dibatalkan tidak dapat diperbarui ke status proses qc.",
		})
	}

	// Update order processing status to "qc process"
	order.ProcessingStatus = "qc_progress"

	if err := oc.DB.Save(&order).Error; err != nil {
		log.Println("QCProcessStatusUpdate - Failed to update order processing status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pemrosesan pesanan",
		})
	}

	// Reload the data with fresh query
	var reloadedOrder models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&reloadedOrder, order.ID).Error; err != nil {
		log.Println("QCProcessStatusUpdate - Failed to load order:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan",
		})
	}

	log.Println("QCProcessStatusUpdate completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Status pemrosesan pesanan berhasil diperbarui ke proses qc",
		Data:    reloadedOrder.ToOrderResponse(),
	})
}

// PickingCompletedStatusUpdate updates the picking completed status of an order
// @Summary Update Picking Completed Status
// @Description Update the picking completed status of an order
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Order}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/{id}/status/picking-completed [put]
func (oc *OrderController) PickingCompletedStatusUpdate(c fiber.Ctx) error {
	log.Println("PickingCompletedStatusUpdate called")
	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("id = ?", id).First(&order).Error; err != nil {
		log.Println("PickingCompletedStatusUpdate - Order not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Order dengan id " + id + " tidak ditemukan.",
		})
	}

	// Check if order processing status allows modification
	if order.ProcessingStatus == "picking_completed" {
		log.Println("PickingCompletedStatusUpdate - Order is already in picking completed status.")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan sudah berada dalam status picking completed.",
		})
	}

	// Check if order is canceled
	if order.EventStatus == "canceled" {
		log.Println("PickingCompletedStatusUpdate - Canceled order cannot be updated to picking completed status.")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan yang dibatalkan tidak dapat diperbarui ke status picking completed.",
		})
	}

	// Update order processing status to "picking_completed"
	order.ProcessingStatus = "picking_completed"
	if err := oc.DB.Save(&order).Error; err != nil {
		log.Println("PickingCompletedStatusUpdate - Failed to update order processing status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pemrosesan pesanan",
		})
	}

	// Reload the data with fresh query
	var reloadedOrder models.Order
	if err := oc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").First(&reloadedOrder, order.ID).Error; err != nil {
		log.Println("PickingCompletedStatusUpdate - Failed to load order:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat pesanan",
		})
	}

	log.Println("PickingCompletedStatusUpdate completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Status pemrosesan pesanan berhasil diperbarui ke picking completed",
		Data:    reloadedOrder.ToOrderResponse(),
	})
}
