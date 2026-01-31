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

type OnlineFlowController struct {
	DB *gorm.DB
}

func NewOnlineFlowController(db *gorm.DB) *OnlineFlowController {
	return &OnlineFlowController{DB: db}
}

// Uniqe response struct for OnlineFlow
type OnlineUserFlowInfo struct {
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

type QcOnlineFlowInfo struct {
	QcBy      string `json:"qcBy,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type OnlineOutboundFlowInfo struct {
	OutboundBy      string `json:"outboundBy,omitempty"`
	Expedition      string `json:"expedition"`
	ExpeditionColor string `json:"expeditionColor"`
	CreatedAt       string `json:"createdAt"`
}

type OnlineOrderFlowInfo struct {
	TrackingNumber   string              `json:"trackingNumber"`
	ProcessingStatus string              `json:"processingStatus"`
	EventStatus      string              `json:"eventStatus"`
	OrderGineeID     string              `json:"orderGineeId"`
	Complained       bool                `json:"complained"`
	CreatedAt        string              `json:"createdAt"`
	AssignedBy       *OnlineUserFlowInfo `json:"assignedBy,omitempty"`
	AssignedAt       *string             `json:"assignedAt,omitempty"`
	PickedBy         *OnlineUserFlowInfo `json:"pickedBy,omitempty"`
	PickedAt         *string             `json:"pickedAt,omitempty"`
	PendingBy        *OnlineUserFlowInfo `json:"pendingBy,omitempty"`
	PendingAt        *string             `json:"pendingAt,omitempty"`
	ChangedBy        *OnlineUserFlowInfo `json:"changedBy,omitempty"`
	ChangedAt        *string             `json:"changedAt,omitempty"`
	DuplicatedBy     *OnlineUserFlowInfo `json:"duplicatedBy,omitempty"`
	DuplicatedAt     *string             `json:"duplicatedAt,omitempty"`
	CancelledBy      *OnlineUserFlowInfo `json:"cancelledBy,omitempty"`
	CancelledAt      *string             `json:"cancelledAt,omitempty"`
}

type OnlineFlowResponse struct {
	TrackingNumber string                  `json:"trackingNumber"`
	QCOnline       *QcOnlineFlowInfo       `json:"qcOnline,omitempty"`
	Outbound       *OnlineOutboundFlowInfo `json:"outbound,omitempty"`
	Order          *OnlineOrderFlowInfo    `json:"order,omitempty"`
}

type OnlineFlowsListResponse struct {
	OnlineFlows []OnlineFlowResponse `json:"onlineFlows"`
}

// Helper function to build OnlineFlowResponse for a given tracking number
func (ofc *OnlineFlowController) BuildOnlineFlow(trackingNumber string) OnlineFlowResponse {
	var response OnlineFlowResponse
	response.TrackingNumber = trackingNumber

	// 1. Query QC Online (Primary Table)
	var qcOnline models.QCOnline
	if err := ofc.DB.Preload("QCOnlineDetails").Preload("QCUser").Where("tracking_number = ?", trackingNumber).First(&qcOnline).Error; err == nil {
		qcBy := ""
		if qcOnline.QCUser != nil {
			qcBy = qcOnline.QCUser.FullName
		}

		response.QCOnline = &QcOnlineFlowInfo{
			QcBy:      qcBy,
			CreatedAt: qcOnline.CreatedAt.Format("02-01-2006 15:04:05"),
		}
	}

	// 2. Query Outbound table
	var outbound models.Outbound
	if err := ofc.DB.Preload("OutboundUser").Where("tracking_number = ?", trackingNumber).First(&outbound).Error; err == nil {
		outboundBy := ""
		if outbound.OutboundUser != nil {
			outboundBy = outbound.OutboundUser.FullName
		}

		response.Outbound = &OnlineOutboundFlowInfo{
			OutboundBy:      outboundBy,
			Expedition:      outbound.Expedition,
			ExpeditionColor: outbound.ExpeditionColor,
			CreatedAt:       outbound.CreatedAt.Format("02-01-2006 15:04:05"),
		}
	}

	// 3. Query Order table
	var order models.Order
	if err := ofc.DB.Preload("OrderDetails").Preload("AssignUser").Preload("PickUser").Preload("PendingUser").Preload("ChangeUser").Preload("DuplicateUser").Preload("CancelUser").Preload("DuplicateUser").Where("tracking_number = ?", trackingNumber).First(&order).Error; err == nil {
		orderInfo := &OnlineOrderFlowInfo{
			TrackingNumber:   order.TrackingNumber,
			ProcessingStatus: order.ProcessingStatus,
			EventStatus:      order.EventStatus,
			OrderGineeID:     order.OrderGineeID,
			Complained:       order.Complained,
			CreatedAt:        order.CreatedAt.Format("02-01-2006 15:04:05"),
		}

		// Processing status human readable
		switch order.ProcessingStatus {
		case "ready_to_pick":
			orderInfo.ProcessingStatus = "Ready to Pick"
		case "picking_progress":
			orderInfo.ProcessingStatus = "Picking in Progress"
		case "picking_pending":
			orderInfo.ProcessingStatus = "Picking is Pending"
		case "picking_completed":
			orderInfo.ProcessingStatus = "Picking Completed"
		case "qc_progress":
			orderInfo.ProcessingStatus = "QC in Progress"
		case "qc_completed":
			orderInfo.ProcessingStatus = "QC Completed"
		case "outbound_completed":
			orderInfo.ProcessingStatus = "Outbound Completed"
		}

		//Event status human readable
		switch order.EventStatus {
		case "in_progress":
			orderInfo.EventStatus = "In Progress"
		case "completed":
			orderInfo.EventStatus = "Completed"
		case "cancelled":
			orderInfo.EventStatus = "Cancelled"
		case "pending":
			orderInfo.EventStatus = "Pending"
		case "duplicated":
			orderInfo.EventStatus = "Duplicated"
		}

		// user visual handlers
		if order.AssignedBy != nil && order.AssignUser != nil {
			orderInfo.AssignedBy = &OnlineUserFlowInfo{
				Username: order.AssignUser.Username,
				FullName: order.AssignUser.FullName,
			}
			if order.AssignedAt != nil {
				assignedAt := order.AssignedAt.Format("02-01-2006 15:04:05")
				orderInfo.AssignedAt = &assignedAt
			}
		}

		if order.PickedBy != nil && order.PickUser != nil {
			orderInfo.PickedBy = &OnlineUserFlowInfo{
				Username: order.PickUser.Username,
				FullName: order.PickUser.FullName,
			}
			if order.PickedAt != nil {
				pickedAt := order.PickedAt.Format("02-01-2006 15:04:05")
				orderInfo.PickedAt = &pickedAt
			}
		}

		if order.PendingBy != nil && order.PendingUser != nil {
			orderInfo.PendingBy = &OnlineUserFlowInfo{
				Username: order.PendingUser.Username,
				FullName: order.PendingUser.FullName,
			}
			if order.PendingAt != nil {
				pendingAt := order.PendingAt.Format("02-01-2006 15:04:05")
				orderInfo.PendingAt = &pendingAt
			}
		}

		if order.ChangedBy != nil && order.ChangeUser != nil {
			orderInfo.ChangedBy = &OnlineUserFlowInfo{
				Username: order.ChangeUser.Username,
				FullName: order.ChangeUser.FullName,
			}
			if order.ChangedAt != nil {
				changedAt := order.ChangedAt.Format("02-01-2006 15:04:05")
				orderInfo.ChangedAt = &changedAt
			}
		}

		if order.CanceledBy != nil && order.CancelUser != nil {
			orderInfo.CancelledBy = &OnlineUserFlowInfo{
				Username: order.CancelUser.Username,
				FullName: order.CancelUser.FullName,
			}
			if order.CanceledAt != nil {
				canceledAt := order.CanceledAt.Format("02-01-2006 15:04:05")
				orderInfo.CancelledAt = &canceledAt
			}
		}

		if order.DuplicatedBy != nil && order.DuplicateUser != nil {
			orderInfo.DuplicatedBy = &OnlineUserFlowInfo{
				Username: order.DuplicateUser.Username,
				FullName: order.DuplicateUser.FullName,
			}
			if order.DuplicatedAt != nil {
				duplicatedAt := order.DuplicatedAt.Format("02-01-2006 15:04:05")
				orderInfo.DuplicatedAt = &duplicatedAt
			}
		}
		response.Order = orderInfo
	}
	return response
}

// GetOnlineFlows retrieves all online flows with their associated QC Online, Outbound, and Order details (with pagination and search)
// @Summary Get all online flows
// @Description Retrieve all online flows with their associated QC Online, Outbound, and Order details (with pagination and search)
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number for pagination" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param startDate query string false "Filter by start date (YYYY-MM-DD format)"
// @Param endDate query string false "Filter by end date (YYYY-MM-DD format)"
// @Param search query string false "Search term for tracking number"
// @Success 200 {array} utils.SuccessPaginatedResponse{data=[]OnlineFlowsListResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/flows [get]
func (ofc *OnlineFlowController) GetOnlineFlows(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var qcOnlines []models.QCOnline

	// Build base query
	query := ofc.DB.Preload("QCOnlineDetails").Preload("QCUser").Model(&models.QCOnline{}).Order("created_at DESC")

	// Date range filter if provided
	startDate := c.Query("startDate", "")
	endDate := c.Query("endDate", "")
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

	// Search condition if provided
	search := c.Query("search", "")
	if search != "" {
		query = query.Where("tracking_number LIKE ?", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Offset(offset).Limit(limit).Find(&qcOnlines).Error; err != nil {
		log.Println("Error retrieving online flows:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data online flows",
		})
	}

	// Format response
	var onlineFlows []OnlineFlowResponse
	for _, qcOnline := range qcOnlines {
		flow := ofc.BuildOnlineFlow(qcOnline.TrackingNumber)
		onlineFlows = append(onlineFlows, flow)
	}

	response := OnlineFlowsListResponse{
		OnlineFlows: onlineFlows,
	}

	// Build success message with all filters
	message := "Online flows retrieved successfully"
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

	log.Println(message)
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    response,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetOnlineFlow retrieves a single online flow by tracking number
// @Summary Get a single online flow
// @Description Retrieve a single online flow by tracking number
// @Tags Onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param trackingNumber path string true "Tracking number of the online flow"
// @Success 200 {object} utils.SuccessResponse{data=OnlineFlowResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/onlines/flows/{trackingNumber} [get]
func (ofc *OnlineFlowController) GetOnlineFlow(c fiber.Ctx) error {
	trackingNumber := c.Params("trackingNumber")

	if trackingNumber == "" {
		log.Println("Tracking number is required")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Nomor pelacakan wajib diisi",
		})
	}

	flow := ofc.BuildOnlineFlow(trackingNumber)

	// Check if qc-online exists
	if flow.QCOnline == nil {
		log.Println("No QC Online found with tracking number:", trackingNumber)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tidak ditemukan QC Online dengan nomor pelacakan yang diberikan",
		})
	}

	log.Println("Online flow retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data online flow berhasil diambil",
		Data:    flow,
	})
}
