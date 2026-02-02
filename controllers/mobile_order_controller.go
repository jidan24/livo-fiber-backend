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

type MobileOrderController struct {
	DB *gorm.DB
}

func NewMobileOrderController(db *gorm.DB) *MobileOrderController {
	return &MobileOrderController{DB: db}
}

// Request structs
type MobileBulkAssignPickerRequest struct {
	PickerID        uint     `json:"pickerId" validate:"required"`
	TrackingNumbers []string `json:"trackingNumbers" validate:"required"`
}

type PendingPickRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type PickedOrderUpdateRequest struct {
	SKU      string `json:"sku" validate:"required"`
	Quantity int    `json:"quantity" validate:"required"`
}

// Unique response structs
type MobileBulkAssignPickerResponse struct {
	Summary        BulkAssignSummary      `json:"summary"`
	AssignedOrders []models.OrderResponse `json:"assignedOrders"`
	SkippedOrders  []SkippedAssignment    `json:"skippedOrders"`
	FailedOrders   []FailedAssignment     `json:"failedOrders"`
}

type BulkAssignSummary struct {
	Total    int `json:"total"`
	Assigned int `json:"assigned"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

type SkippedAssignment struct {
	Index          int    `json:"index"`
	TrackingNumber string `json:"tracking"`
	Reason         string `json:"reason"`
}

type FailedAssignment struct {
	Index          int    `json:"index"`
	TrackingNumber string `json:"tracking"`
	Error          string `json:"error"`
}

// GetMyPickingOrders retrieves all orders assigned to a picker
// @Summary Get My Picking Orders
// @Description Retrieve all orders assigned to a picker
// @Tags Mobile Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.SuccessResponse{data=[]models.OrderResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-orders/my-picking-orders [get]
func (moc *MobileOrderController) GetMyPickingOrders(c fiber.Ctx) error {
	log.Println("GetMyPickingOrders called")
	var orders []models.Order

	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("GetMyPickingOrders - Invalid user ID:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Base query to get orders assigned to the picker
	query := moc.DB.Model(&models.Order{}).Preload("OrderDetails").Preload("PickUser").Preload("AssignUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").
		Where("picked_by = ? AND processing_status = ?", userID, "picking_progress").Order("created_at DESC").Find(&orders)

	// Get total count
	var total int64
	query.Count(&total)

	// Load product details in order responses
	for i := range orders {
		for j := range orders[i].OrderDetails {
			var product models.Product
			if err := moc.DB.Where("sku = ?", orders[i].OrderDetails[j].SKU).First(&product).Error; err == nil {
				orders[i].OrderDetails[j].Product = &product
			}
		}
	}

	// Include product details in order responses
	orderResponses := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		orderResp := *order.ToOrderResponse()
		orderResponses[i] = orderResp
	}

	log.Println("GetMyPickingOrders completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessTotaledResponse{
		Success: true,
		Message: "Data picking order berhasil diambil",
		Data:    orderResponses,
		Total:   total,
	})
}

// GetMyPickingOrder retrieves a specific order assigned to a picker
// @Summary Get My Picking Order
// @Description Retrieve a specific order assigned to a picker
// @Tags Mobile Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.OrderResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-orders/my-picking-orders/{id} [get]
func (moc *MobileOrderController) GetMyPickingOrder(c fiber.Ctx) error {
	log.Println("GetMyPickingOrder called")
	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("GetMyPickingOrder - Invalid user ID:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Parse id parameter
	id := c.Params("id")
	var order models.Order

	if err := moc.DB.Model(&models.Order{}).Preload("OrderDetails").Preload("PickUser").Preload("AssignUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").
		Where("id = ?", id).Where("picked_by = ?", userID).First(&order).First(&order).Error; err != nil {
		log.Println("GetMyPickingOrder - Order not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan tidak ditemukan",
		})
	}

	// load product details in order response
	for i := range order.OrderDetails {
		var product models.Product
		if err := moc.DB.Where("sku = ?", order.OrderDetails[i].SKU).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	log.Println("GetMyPickingOrder completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil diambil",
		Data:    order.ToOrderResponse(),
	})
}

// CompletePickingOrder marks an order as picked by the picker
// @Summary Complete Picking Order
// @Description Mark an order as picked by the picker
// @Tags Mobile Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.OrderResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-orders/my-picking-orders/{id}/complete [put]
func (moc *MobileOrderController) CompletePickingOrder(c fiber.Ctx) error {
	log.Println("CompletePickingOrder called")
	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("CompletePickingOrder - Invalid user ID:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := moc.DB.Preload("OrderDetails").Where("id = ?", id).Where("picked_by = ?", userID).First(&order).Error; err != nil {
		log.Println("CompletePickingOrder - Order not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesan dengan id " + id + " tidak ditemukan.",
		})
	}

	// Check if isPicked is true for all order details
	for _, detail := range order.OrderDetails {
		if !detail.IsPicked {
			log.Println("CompletePickingOrder - Order details not all picked")
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Not all order details are picked",
			})
		}
	}

	// Start transaction
	tx := moc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update order status to picking completed
	now := time.Now()
	userIDUint := uint(userID)
	order.PickedBy = &userIDUint
	order.PickedAt = &now
	order.ProcessingStatus = "picking_completed"

	if err := tx.Save(&order).Error; err != nil {
		log.Println("CompletePickingOrder - Failed to update order status:", err)
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pesanan: " + err.Error(),
		})
	}

	// Create picked order log
	pickedOrder := models.PickedOrder{
		OrderID:  order.ID,
		PickedBy: uint(userID),
	}

	if err := tx.Create(&pickedOrder).Error; err != nil {
		log.Println("CompletePickingOrder - Failed to create picked order log:", err)
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat log picking order: " + err.Error(),
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Println("CompletePickingOrder - Failed to commit transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal melakukan commit transaksi: " + err.Error(),
		})
	}

	// Reload order with updated data
	if err := moc.DB.Preload("OrderDetails").Preload("PickUser").Preload("AssignUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").
		Where("id = ?", order.ID).First(&order).Where("picked_by = ?", userID).Error; err != nil {
		log.Println("CompletePickingOrder - Failed to reload order data:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat ulang data pesanan: " + err.Error(),
		})
	}

	// load product details in order response
	for i := range order.OrderDetails {
		var product models.Product
		if err := moc.DB.Where("sku = ?", order.OrderDetails[i].SKU).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	log.Println("CompletePickingOrder completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil ditandai sebagai picked",
		Data:    order.ToOrderResponse(),
	})
}

// PendingPickOrder marks an order as pending pick by the picker
// @Summary Pending Pick Order
// @Description Mark an order as pending pick by the picker
// @Tags Mobile Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param request body PendingPickRequest true "Pending pick request with coordinator credentials"
// @Success 200 {object} utils.SuccessResponse{data=models.OrderResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-orders/my-picking-orders/{id}/pending [put]
func (moc *MobileOrderController) PendingPickOrder(c fiber.Ctx) error {
	log.Println("PendingPickOrder called")
	//Param id parameter
	id := c.Params("id")
	var order models.Order
	if err := moc.DB.Where("id = ?", id).First(&order).Error; err != nil {
		log.Println("PendingPickOrder - Order not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan dengan id " + id + " tidak ditemukan.",
		})
	}

	// Parse request body
	var req PendingPickRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("PendingPickOrder - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Check if status is picking process
	if order.ProcessingStatus != "picking_progress" {
		log.Println("PendingPickOrder - Order not in picking progress status")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan tidak dalam status poses picking",
		})
	}

	// Authenticate coordinator credentials
	var coordinator models.User
	if err := moc.DB.Preload("Roles").Where("username = ?", req.Username).First(&coordinator).Error; err != nil {
		log.Println("PendingPickOrder - Invalid coordinator credentials:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Kredensial koordinator tidak valid",
		})
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, coordinator.Password) {
		log.Println("PendingPickOrder - Invalid coordinator password")
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Kredensial koordinator tidak valid",
		})
	}

	// Check if user has developer, coordinator or superadmin role
	hasCoordinatorRole := false
	for _, role := range coordinator.Roles {
		if role.RoleName == "developer" || role.RoleName == "coordinator" || role.RoleName == "superadmin" {
			hasCoordinatorRole = true
			break
		}
	}

	if !hasCoordinatorRole {
		log.Println("PendingPickOrder - User does not have required permissions")
		return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak memiliki izin yang diperlukan",
		})
	}

	// Get current developer, coordinator, or superadmin user ID that credentials belong to
	coordinatorID := coordinator.ID

	// Update order status to pending pick
	now := time.Now()
	order.PendingBy = &coordinatorID
	order.PendingAt = &now
	order.ProcessingStatus = "picking_pending"

	if err := moc.DB.Save(&order).Error; err != nil {
		log.Println("PendingPickOrder - Failed to update order status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui status pesanan: " + err.Error(),
		})
	}

	// Reload order with updated data
	if err := moc.DB.Preload("OrderDetails").Preload("PickUser").Preload("AssignUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").
		Where("id = ?", order.ID).First(&order).Error; err != nil {
		log.Println("PendingPickOrder - Failed to reload order data:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat ulang data pesanan: " + err.Error(),
		})
	}

	// load product details in order response
	for i := range order.OrderDetails {
		var product models.Product
		if err := moc.DB.Where("sku = ?", order.OrderDetails[i].SKU).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	log.Println("PendingPickOrder completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan berhasil ditandai sebagai pending pick",
		Data:    order.ToOrderResponse(),
	})
}

// BulkAssignPicker handles bulk assignment of orders to a picker
// @Summary Bulk Assign Picker
// @Description Bulk assign orders to a picker
// @Tags Mobile Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MobileBulkAssignPickerRequest true "Bulk assign request with picker ID and tracking numbers"
// @Success 200 {object} utils.SuccessResponse{data=MobileBulkAssignPickerResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-orders/bulk-assign-picker [put]
func (moc *MobileOrderController) BulkAssignPicker(c fiber.Ctx) error {
	log.Println("BulkAssignPicker called")
	// Parse request body
	var req MobileBulkAssignPickerRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("BulkAssignPicker - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Get current user from context
	assignerIDStr := c.Locals("userId").(string)
	assignerID, err := strconv.ParseUint(assignerIDStr, 10, 32)
	if err != nil {
		log.Println("BulkAssignPicker - Invalid user ID:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Verify if the picker exists
	var picker models.User
	if err := moc.DB.Where("id = ?", req.PickerID).First(&picker).Error; err != nil {
		log.Println("BulkAssignPicker - Picker not found:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Picker dengan ID " + strconv.FormatUint(uint64(req.PickerID), 10) + " tidak ditemukan",
		})
	}

	var assignedOrders []models.OrderResponse
	var skippedOrders []SkippedAssignment
	var failedOrders []FailedAssignment

	assignerIDUint := uint(assignerID)
	now := time.Now()

	// Process each tracking number
	for i, trackingNumber := range req.TrackingNumbers {
		var order models.Order
		// Find order by tracking number
		if err := moc.DB.Where("tracking_number = ?", trackingNumber).First(&order).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				skippedOrders = append(skippedOrders, SkippedAssignment{
					Index:          i,
					TrackingNumber: trackingNumber,
					Reason:         "Order not found",
				})
			} else {
				failedOrders = append(failedOrders, FailedAssignment{
					Index:          i,
					TrackingNumber: trackingNumber,
					Error:          "Failed to find order",
				})
			}
			continue
		}

		// Validate order status
		if order.EventStatus == "canceled" {
			skippedOrders = append(skippedOrders, SkippedAssignment{
				Index:          i,
				TrackingNumber: trackingNumber,
				Reason:         "Order is canceled",
			})
			continue
		}

		// Only allow assignment if order is "ready to pick" or "pending picking"
		if order.ProcessingStatus != "ready_to_pick" && order.ProcessingStatus != "picking_pending" {
			skippedOrders = append(skippedOrders, SkippedAssignment{
				Index:          i,
				TrackingNumber: trackingNumber,
				Reason:         "Order not in assignable status",
			})
			continue
		}

		// Update order with picker assignment
		order.PickedBy = &req.PickerID
		order.AssignedAt = &now
		order.AssignedBy = &assignerIDUint
		order.ProcessingStatus = "picking_progress"

		if err := moc.DB.Save(&order).Error; err != nil {
			failedOrders = append(failedOrders, FailedAssignment{
				Index:          i,
				TrackingNumber: trackingNumber,
				Error:          err.Error(),
			})
			continue
		}

		// Load order details for response
		if err := moc.DB.Preload("OrderDetails").Preload("PickUser").Preload("AssignUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").
			Where("id = ?", order.ID).First(&order).Error; err != nil {
			failedOrders = append(failedOrders, FailedAssignment{
				Index:          i,
				TrackingNumber: trackingNumber,
				Error:          "Failed to load order details",
			})
			continue
		}

		// load product details in order response
		for i := range order.OrderDetails {
			var product models.Product
			if err := moc.DB.Where("sku = ?", order.OrderDetails[i].SKU).First(&product).Error; err == nil {
				order.OrderDetails[i].Product = &product
			}
		}

		assignedOrders = append(assignedOrders, *order.ToOrderResponse())
	}

	// Prepare summary
	response := MobileBulkAssignPickerResponse{
		Summary: BulkAssignSummary{
			Total:    len(req.TrackingNumbers),
			Assigned: len(assignedOrders),
			Skipped:  len(skippedOrders),
			Failed:   len(failedOrders),
		},
		AssignedOrders: assignedOrders,
		SkippedOrders:  skippedOrders,
		FailedOrders:   failedOrders,
	}

	// Determine response status and message
	statusCode := fiber.StatusOK
	message := "Bulk picker assignment completed"

	if len(assignedOrders) == 0 {
		if len(skippedOrders) > 0 {
			message = "All orders were skipped"
		} else {
			statusCode = fiber.ErrBadRequest.Code
			message = "No orders could be assigned"
		}
	} else if len(failedOrders) > 0 || len(skippedOrders) > 0 {
		message = "Bulk picker assignment completed with some issues"
	} else {
		message = fmt.Sprintf("Successfully assigned %d order(s) to picker", len(assignedOrders))
	}

	log.Printf("BulkAssignPicker completed (assigned=%d, skipped=%d, failed=%d)\n", len(assignedOrders), len(skippedOrders), len(failedOrders))
	return c.Status(statusCode).JSON(utils.SuccessResponse{
		Success: true,
		Message: message,
		Data:    response,
	})
}

// GetMobilePickedOrders retrieves all picked orders for coordinator by mobile
// @Summary Get Mobile Picked Orders
// @Description Retrieve all picked orders for coordinator by mobile
// @Tags Mobile Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of orders per page" default(10)
// @Param search query string false "Search term for tracking number or ginee order ID"
// @Success 200 {object} utils.SuccessTotaledResponse{data=[]models.OrderResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-orders [get]
func (moc *MobileOrderController) GetMobilePickedOrders(c fiber.Ctx) error {
	log.Println("GetMobilePickedOrders called")
	// Get current logged in user from context
	userIDStr := c.Locals("userId").(string)
	UserID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("GetMobilePickedOrders - Invalid user ID:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var pickedOrders []models.Order

	// Base query to get picked orders
	query := moc.DB.Model(&models.Order{}).Preload("OrderDetails").Preload("PickUser").Preload("AssignUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("processing_status = ?", "picking_progress").Where("assigned_by = ?", UserID)

	// Apply search filter if provided
	search := c.Query("search", "")
	if search != "" {
		query = query.Where("orders.tracking_number ILIKE ? OR orders.order_ginee_id ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	query = query.Order("created_at DESC")
	if err := query.Offset(offset).Limit(limit).Find(&pickedOrders).Error; err != nil {
		log.Println("GetMobilePickedOrders - Failed to retrieve picked orders:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data picked order",
		})
	}

	// Load product details in order responses
	for i := range pickedOrders {
		for j := range pickedOrders[i].OrderDetails {
			var product models.Product
			if err := moc.DB.Where("sku = ?", pickedOrders[i].OrderDetails[j].SKU).First(&product).Error; err == nil {
				pickedOrders[i].OrderDetails[j].Product = &product
			}
		}
	}

	// Format response
	pickedOrderList := make([]models.OrderResponse, len(pickedOrders))
	for i, pickedOrder := range pickedOrders {
		pickedOrderList[i] = *pickedOrder.ToOrderResponse()
	}

	// Build success message
	message := "Picked orders retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	log.Println("GetMobilePickedOrders completed successfully")
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

// GetMobilePickedOrder retrieves a specific picked order by ID
// @Summary Get Mobile Picked Order
// @Description Retrieve a specific picked order by ID
// @Tags Mobile Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Picked Order ID"
// @Success 200 {object} utils.SuccessResponse{data=models.OrderResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-orders/{id} [get]
func (moc *MobileOrderController) GetMobilePickedOrder(c fiber.Ctx) error {
	log.Println("GetMobilePickedOrder called")
	// Parse id parameter
	id := c.Params("id")
	var pickedOrder models.Order
	if err := moc.DB.Preload("OrderDetails").Preload("PickUser").Preload("PickUser").Preload("AssignUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Where("id = ?", id).First(&pickedOrder).Error; err != nil {
		log.Println("GetMobilePickedOrder - Picked order not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Picked order dengan id " + id + " tidak ditemukan.",
		})
	}

	// Load product details in order response
	for i := range pickedOrder.OrderDetails {
		var product models.Product
		if err := moc.DB.Where("sku = ?", pickedOrder.OrderDetails[i].SKU).First(&product).Error; err == nil {
			pickedOrder.OrderDetails[i].Product = &product
		}
	}

	log.Println("GetMobilePickedOrder completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data picked order berhasil diambil",
		Data:    pickedOrder.ToOrderResponse(),
	})
}

// UpdatePickedOrder update IsPicked status of an order detail verified by sku and quantity of detail order
// @Summary Update Picked Order Detail
// @Description Update IsPicked status of an order detail verified by sku
// @Tags Mobile Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param request body PickedOrderUpdateRequest true "Picked order detail update request with SKU and Quantity"
// @Success 200 {object} utils.SuccessResponse{data=models.OrderResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-orders/my-picking-orders/{id}/picked [put]
func (moc *MobileOrderController) UpdatePickedOrder(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var order models.Order
	if err := moc.DB.Preload("OrderDetails").Where("id = ?", id).First(&order).Error; err != nil {
		log.Println("UpdatePickedOrder - Order not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pesanan dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req PickedOrderUpdateRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("UpdatePickedOrder - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi data permintaan tidak valid",
		})
	}

	// Check SKU, quantity, and isPicked status one by one
	var matchedIndex int = -1

	for i := range order.OrderDetails {
		if order.OrderDetails[i].SKU == req.SKU {
			matchedIndex = i
			break
		}
	}

	if matchedIndex == -1 {
		log.Println("UpdatePickedOrder - SKU not found in order")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "SKU '" + req.SKU + "' Data tidak ditemukan dalam pesanan ini",
		})
	}

	if order.OrderDetails[matchedIndex].Quantity != req.Quantity {
		log.Printf("UpdatePickedOrder - Quantity mismatch for SKU %s (expected: %d, received: %d)", req.SKU, order.OrderDetails[matchedIndex].Quantity, req.Quantity)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Jumlah tidak sesuai untuk SKU '%s' (seharusnya: %d, diterima: %d)", req.SKU, order.OrderDetails[matchedIndex].Quantity, req.Quantity),
		})
	}

	if order.OrderDetails[matchedIndex].IsPicked {
		log.Printf("UpdatePickedOrder - Order detail already picked (SKU: %s, IsPicked: %v)", req.SKU, order.OrderDetails[matchedIndex].IsPicked)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Detail pesanan dengan SKU '" + req.SKU + "' sudah di proses",
		})
	}

	// Update IsPicked to true by directly updating the order detail record
	if err := moc.DB.Model(&order.OrderDetails[matchedIndex]).Update("is_picked", true).Error; err != nil {
		log.Println("UpdatePickedOrder - Failed to update order detail:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui detail pesanan",
		})
	}

	// Reload the order with fresh data
	if err := moc.DB.Preload("OrderDetails").Where("id = ?", id).First(&order).Error; err != nil {
		log.Println("UpdatePickedOrder - Failed to reload order:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memuat ulang data pesanan",
		})
	}

	log.Println("UpdatePickedOrder completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pesanan yang di-pick berhasil diperbarui",
		Data:    order.ToOrderResponse(),
	})
}
