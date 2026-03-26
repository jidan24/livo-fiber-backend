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

type ReturnController struct {
	DB *gorm.DB
}

func NewReturnController(db *gorm.DB) *ReturnController {
	return &ReturnController{DB: db}
}

// Request structs
type CreateReturnRequest struct {
	NewTrackingNumber string  `json:"newTrackingNumber" validate:"required"`
	ChannelID         uint    `json:"channelId" validate:"required"`
	StoreID           uint    `json:"storeId" validate:"required"`
	TrackingNumber    *string `json:"trackingNumber,omitempty"`
	ReturnType        *string `json:"returnType,omitempty"`
	ReturnReason      *string `json:"returnReason,omitempty"`
	ReturnNumber      *string `json:"returnNumber,omitempty"`
	ScrapNumber       *string `json:"scrapNumber,omitempty"`
}

type UpdateReturnRequest struct {
	TrackingNumber *string `json:"trackingNumber,omitempty"`
	ReturnType     *string `json:"returnType,omitempty"`
	ReturnReason   *string `json:"returnReason,omitempty"`
	ReturnNumber   *string `json:"returnNumber,omitempty"`
	ScrapNumber    *string `json:"scrapNumber,omitempty"`
}

// GetReturns retrieves a list of returns with pagination and search
// @Summary Get Returns
// @Description Retrieve a list of returns with pagination and search
// @Tags Returns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of returns per page" default(10)
// @Param search query string false "Search term for new tracking number, order ginee ID, tracking number"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.ReturnResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/returns [get]
func (rc *ReturnController) GetReturns(c fiber.Ctx) error {
	log.Println("GetReturns called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var returns []models.Return

	// Build base query
	query := rc.DB.Preload("ReturnDetails").Preload("Channel").Preload("Store").Preload("CreateUser").Preload("UpdateUser").Model(&models.Return{}).Order("created_at DESC")

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
		query = query.Where("new_tracking_number ILIKE ? OR order_ginee_id ILIKE ? OR tracking_number ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Offset(offset).Limit(limit).Find(&returns).Error; err != nil {
		log.Println("GetReturns - Failed to retrieve returns:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data retur",
		})
	}

	// Include Order details for each return if TrackingNumber exists in Order
	for i := range returns {
		if returns[i].TrackingNumber != nil {
			var order models.Order
			if err := rc.DB.Where("tracking_number = ?", returns[i].TrackingNumber).First(&order).Error; err == nil {
				returns[i].Order = &order
			}
		}

		// Load products for return details
		if returns[i].ReturnDetails != nil {
			for j := range *returns[i].ReturnDetails {
				detail := &(*returns[i].ReturnDetails)[j]
				if detail.ProductSKU != nil {
					var product models.Product
					if err := rc.DB.Where("sku = ?", *detail.ProductSKU).First(&product).Error; err == nil {
						detail.Product = &product
					}
				}
			}
		}
	}

	// Format response
	returnList := make([]models.ReturnResponse, len(returns))
	for i, ret := range returns {
		returnList[i] = ret.ToResponse()
	}

	// Build success message
	message := "Returns retrieved successfully"
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
	log.Println("GetReturns completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    returnList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetReturn retrieves a single return by ID
// @Summary Get Return
// @Description Retrieve a single return by ID
// @Tags Returns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Return ID"
// @Success 200 {object} utils.SuccessResponse{data=models.ReturnResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/returns/{id} [get]
func (rc *ReturnController) GetReturn(c fiber.Ctx) error {
	log.Println("GetReturn called")
	// Parse id parameters
	id := c.Params("id")
	var ret models.Return
	if err := rc.DB.Preload("ReturnDetails").Preload("CreateUser").Preload("UpdateUser").First(&ret, id).Error; err != nil {
		log.Println("GetReturn - Return not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Return dengan id " + id + " tidak ditemukan",
		})
	}

	// Include Order details if TrackingNumber exists in Order
	if ret.TrackingNumber != nil {
		var order models.Order
		if err := rc.DB.Where("tracking_number = ?", ret.TrackingNumber).First(&order).Error; err == nil {
			ret.Order = &order
		}
	}

	// Load products for return details
	if ret.ReturnDetails != nil {
		for i := range *ret.ReturnDetails {
			detail := &(*ret.ReturnDetails)[i]
			if detail.ProductSKU != nil {
				var product models.Product
				if err := rc.DB.Where("sku = ?", *detail.ProductSKU).First(&product).Error; err == nil {
					detail.Product = &product
				}
			}
		}
	}

	// Return success response
	log.Println("GetReturn completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data retur berhasil diambil",
		Data:    ret.ToResponse(),
	})
}

// CreateReturn handles the creation of a new return with details from order details table
// @Summary Create Return
// @Description Create a new return
// @Tags Returns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateReturnRequest true "Return details"
// @Success 201 {object} utils.SuccessResponse{data=models.Return}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/returns [post]
func (rc *ReturnController) CreateReturn(c fiber.Ctx) error {
	log.Println("CreateReturn called")
	// Binding request body
	var req CreateReturnRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("CreateReturn - Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Get current user logged in user
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Convert new tracking number to uppercase and trim spaces
	req.NewTrackingNumber = strings.ToUpper(strings.TrimSpace(req.NewTrackingNumber))

	// Check for duplicate NewTrackingNumber
	var existingReturn models.Return
	if err := rc.DB.Where("new_tracking_number = ?", req.NewTrackingNumber).First(&existingReturn).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Retur dengan nomor pelacakan baru " + req.NewTrackingNumber + " sudah terdaftar",
		})
	}

	// If TrackingNumber is provided, convert to uppercase and trim spaces
	var order *models.Order
	if req.TrackingNumber != nil && *req.TrackingNumber != "" {
		tracking := strings.ToUpper(strings.TrimSpace(*req.TrackingNumber))
		req.TrackingNumber = &tracking

		// Check if it exists in Order
		var orderData models.Order
		if err := rc.DB.Preload("OrderDetails").Where("tracking_number = ?", req.TrackingNumber).First(&orderData).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Pesanan dengan nomor pelacakan " + *req.TrackingNumber + " tidak ditemukan",
			})
		}
		order = &orderData
	}

	// Start database transaction
	tx := rc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	ret := models.Return{
		NewTrackingNumber: req.NewTrackingNumber,
		TrackingNumber:    req.TrackingNumber,
		ChannelID:         req.ChannelID,
		StoreID:           req.StoreID,
		ReturnType:        req.ReturnType,
		ReturnReason:      req.ReturnReason,
		ReturnNumber:      req.ReturnNumber,
		ScrapNumber:       req.ScrapNumber,
		CreatedBy:         uint(userID),
	}

	// Set OrderGineeID if order exists
	if order != nil {
		ret.OrderGineeID = &order.OrderGineeID
	}

	if err := tx.Create(&ret).Error; err != nil {
		log.Println("CreateReturn - Failed to create return:", err)
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat retur",
		})
	}

	// Create ReturnDetails from OrderDetails if order exists
	if order != nil {
		for _, orderDetail := range order.OrderDetails {
			returnDetail := models.ReturnDetail{
				ReturnID:   &ret.ID,
				ProductSKU: &orderDetail.SKU,
				Quantity:   &orderDetail.Quantity,
				Price:      &orderDetail.Price,
			}

			if err := tx.Create(&returnDetail).Error; err != nil {
				log.Println("CreateReturn - Failed to create return details:", err)
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal membuat detail retur",
				})
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Println("CreateReturn - Failed to commit transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal melakukan commit transaksi",
		})
	}

	// Reload return with details
	if err := rc.DB.Preload("ReturnDetails").Preload("Channel").Preload("Store").Preload("CreateUser").Preload("UpdateUser").First(&ret, ret.ID).Error; err != nil {
		log.Println("CreateReturn - Failed to retrieve created return:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data retur yang baru dibuat",
		})
	}

	// Include Order details if TrackingNumber exists in Order
	if ret.TrackingNumber != nil {
		var order models.Order
		if err := rc.DB.Where("tracking_number = ?", ret.TrackingNumber).First(&order).Error; err == nil {
			ret.Order = &order
		}
	}

	// Load products for return details
	if ret.ReturnDetails != nil {
		for i := range *ret.ReturnDetails {
			detail := &(*ret.ReturnDetails)[i]
			if detail.ProductSKU != nil {
				var product models.Product
				if err := rc.DB.Where("sku = ?", *detail.ProductSKU).First(&product).Error; err == nil {
					detail.Product = &product
				}
			}
		}
	}

	// Return success response
	log.Println("CreateReturn completed successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Retur berhasil dibuat",
		Data:    ret.ToResponse(),
	})
}

// UpdateReturn handles updating an existing return and if details still empty, populate from order details
// @Summary Update Return
// @Description Update an existing return
// @Tags Returns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Return ID"
// @Param request body UpdateReturnRequest true "Return details to update"
// @Success 200 {object} utils.SuccessResponse{data=models.ReturnResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/returns/{id} [put]
func (rc *ReturnController) UpdateReturn(c fiber.Ctx) error {
	log.Println("UpdateReturn called")
	// Parse id parameters
	id := c.Params("id")
	var ret models.Return
	if err := rc.DB.Preload("ReturnDetails").First(&ret, id).Error; err != nil {
		log.Println("UpdateReturn - Return not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Return dengan id " + id + " tidak ditemukan",
		})
	}

	// Binding request body
	var req UpdateReturnRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Get current user logged in user
	userIDStr := c.Locals("userId").(string)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID pengguna tidak valid",
		})
	}

	// Check if TrackingNumber is provided, convert to uppercase and trim spaces and check if it's exists in Order
	if req.TrackingNumber != nil {
		tracking := strings.ToUpper(strings.TrimSpace(*req.TrackingNumber))
		req.TrackingNumber = &tracking
	}

	// Check if TrackingNumber exists in Order
	var order models.Order
	if req.TrackingNumber != nil {
		if err := rc.DB.Preload("OrderDetails").Where("tracking_number = ?", req.TrackingNumber).First(&order).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Pesanan dangan nomor pelacakan " + *req.TrackingNumber + " tidak ditemukan",
			})
		}
	}

	// Check if TrackingNumber is already exists in another Return
	if req.TrackingNumber != nil {
		var existingReturn models.Return
		if err := rc.DB.Where("tracking_number = ? AND id <> ?", req.TrackingNumber, ret.ID).First(&existingReturn).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Retur dengan nomor pelacakan " + *req.TrackingNumber + " sudah terdaftar",
			})
		}
	}

	// Check for return details before transaction
	needToPopulateDetails := len(*ret.ReturnDetails) == 0 && req.TrackingNumber != nil

	// Start database transaction
	tx := rc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update return fields
	updatedBy := uint(userID)

	ret.TrackingNumber = req.TrackingNumber
	ret.ReturnType = req.ReturnType
	ret.ReturnReason = req.ReturnReason
	ret.ReturnNumber = req.ReturnNumber
	ret.ScrapNumber = req.ScrapNumber
	ret.UpdatedBy = &updatedBy

	// Update OrderGineeID if tracking number is provided and order exists
	if req.TrackingNumber != nil && order.ID != 0 {
		ret.OrderGineeID = &order.OrderGineeID
	}

	if err := tx.Save(&ret).Error; err != nil {
		log.Println("UpdateReturn - Failed to update return:", err)
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui retur",
		})
	}

	// Populate ReturnDetails from OrderDetails if needed
	if needToPopulateDetails {
		for _, orderDetail := range order.OrderDetails {
			returnDetail := models.ReturnDetail{
				ReturnID:   &ret.ID,
				ProductSKU: &orderDetail.SKU,
				Quantity:   &orderDetail.Quantity,
				Price:      &orderDetail.Price,
			}

			if err := tx.Create(&returnDetail).Error; err != nil {
				log.Println("UpdateReturn - Failed to create return details:", err)
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
					Success: false,
					Error:   "Gagal membuat detail retur",
				})
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Println("UpdateReturn - Failed to commit transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal melakukan commit transaksi",
		})
	}

	// Reload return with details
	if err := rc.DB.Preload("ReturnDetails").Preload("Channel").Preload("Store").Preload("CreateUser").Preload("UpdateUser").First(&ret, ret.ID).Error; err != nil {
		log.Println("UpdateReturn - Failed to retrieve updated return:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data retur",
		})
	}

	// Include Order details if TrackingNumber exists in Order
	if ret.TrackingNumber != nil {
		var order models.Order
		if err := rc.DB.Where("tracking_number = ?", ret.TrackingNumber).First(&order).Error; err == nil {
			ret.Order = &order
		}
	}

	// Load products for return details
	if ret.ReturnDetails != nil {
		for i := range *ret.ReturnDetails {
			detail := &(*ret.ReturnDetails)[i]
			if detail.ProductSKU != nil {
				var product models.Product
				if err := rc.DB.Where("sku = ?", *detail.ProductSKU).First(&product).Error; err == nil {
					detail.Product = &product
				}
			}
		}
	}

	// Return success response
	log.Println("UpdateReturn completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Retur berhasil diperbarui",
		Data:    ret.ToResponse(),
	})
}
