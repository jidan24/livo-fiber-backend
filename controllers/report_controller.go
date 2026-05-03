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

type ReportController struct {
	DB *gorm.DB
}

func NewReportController(db *gorm.DB) *ReportController {
	return &ReportController{DB: db}
}

// Unique response structs
type BoxUsageDetail struct {
	TrackingNumber string `json:"trackingNumber"`
	OrderGineeID   string `json:"orderGineeId"`
	BoxName        string `json:"boxName"`
	Quantity       int    `json:"quantity"`
	QcBy           string `json:"qcBy"`
	CreatedAt      string `json:"createdAt"`
	Source         string `json:"source"`
}

type BoxCountReport struct {
	BoxID       uint             `json:"boxId"`
	BoxCode     string           `json:"boxCode"`
	BoxName     string           `json:"boxName"`
	TotalCount  int              `json:"totalCount"`
	RibbonCount int              `json:"ribbonCount"`
	OnlineCount int              `json:"onlineCount"`
	Details     []BoxUsageDetail `json:"details" gorm:"-"`
}

type BoxCountReportsListResponse struct {
	Reports []BoxCountReport `json:"reports"`
}

type OutboundReportsListResponse struct {
	Outbounds []models.OutboundResponse `json:"outbounds"`
}

type ReturnReportsListResponse struct {
	Returns []models.ReturnResponse `json:"returns"`
}

type ComplaintReportsListResponse struct {
	Complaints []models.ComplainResponse `json:"complains"`
}

type ComplainDetailInReport struct {
	ComplainID        uint   `json:"complainId"`
	ComplainCode      string `json:"complainCode"`
	Tracking          string `json:"tracking"`
	OrderGineeID      string `json:"orderGineeId"`
	FeeCharge         uint   `json:"feeCharge"`
	ComplainUpdatedAt string `json:"complainUpdatedAt"`
}

type UserFeeReportWithDetails struct {
	UserID          uint                     `json:"userId"`
	Username        string                   `json:"username"`
	FullName        string                   `json:"fullName"`
	Email           string                   `json:"email"`
	TotalComplaints int                      `json:"totalComplaints"`
	TotalFeeCharge  uint                     `json:"totalFeeCharge"`
	ComplainDetails []ComplainDetailInReport `json:"complainDetails"`
}

type UserFeeReportsWithDetailsListResponse struct {
	Reports []UserFeeReportWithDetails `json:"reports"`
}

// BuildBoxUsageDetails retrieves detailed usage for a specific box
func (rc *ReportController) BuildBoxUsageDetails(boxID uint, startDate, endDate string) []BoxUsageDetail {
	log.Println("BuildBoxUsageDetails called")
	details := []BoxUsageDetail{}

	// Query from QCRibbonDetail with joins
	type RibbonResult struct {
		TrackingNumber string
		OrderGineeID   string
		BoxName        string
		Quantity       int
		FullName       string
		CreatedAt      time.Time
	}

	var ribbonResults []RibbonResult
	ribbonQuery := rc.DB.Table("qc_ribbon_details").
		Select("qc_ribbons.tracking_number, orders.order_ginee_id, boxes.box_name, qc_ribbon_details.quantity, users.full_name, qc_ribbons.created_at").
		Joins("LEFT JOIN qc_ribbons ON qc_ribbons.id = qc_ribbon_details.qc_ribbon_id").
		Joins("LEFT JOIN boxes ON boxes.id = qc_ribbon_details.box_id").
		Joins("LEFT JOIN users ON users.id = qc_ribbons.qc_by").
		Joins("LEFT JOIN orders ON orders.tracking_number = qc_ribbons.tracking_number").
		Where("qc_ribbon_details.box_id = ?", boxID)

	// Apply date filters for ribbon
	if startDate != "" {
		ribbonQuery = ribbonQuery.Where("qc_ribbons.created_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		ribbonQuery = ribbonQuery.Where("qc_ribbons.created_at <= ?", endDate+" 23:59:59")
	}

	ribbonQuery.Scan(&ribbonResults)

	// Add ribbon results to details
	for _, r := range ribbonResults {
		details = append(details, BoxUsageDetail{
			TrackingNumber: r.TrackingNumber,
			OrderGineeID:   r.OrderGineeID,
			BoxName:        r.BoxName,
			Quantity:       r.Quantity,
			QcBy:           r.FullName,
			CreatedAt:      r.CreatedAt.Format("02-01-2006 15:04:05"),
			Source:         "ribbon",
		})
	}

	// Query from QCOnlineDetail with joins
	type OnlineResult struct {
		TrackingNumber string
		OrderGineeID   string
		BoxName        string
		Quantity       int
		FullName       string
		CreatedAt      time.Time
	}

	var onlineResults []OnlineResult
	onlineQuery := rc.DB.Table("qc_online_details").
		Select("qc_onlines.tracking_number, orders.order_ginee_id, boxes.box_name, qc_online_details.quantity, users.full_name, qc_onlines.created_at").
		Joins("LEFT JOIN qc_onlines ON qc_onlines.id = qc_online_details.qc_online_id").
		Joins("LEFT JOIN boxes ON boxes.id = qc_online_details.box_id").
		Joins("LEFT JOIN users ON users.id = qc_onlines.qc_by").
		Joins("LEFT JOIN orders ON orders.tracking_number = qc_onlines.tracking_number").
		Where("qc_online_details.box_id = ?", boxID)

	// Apply date filters for online
	if startDate != "" {
		onlineQuery = onlineQuery.Where("qc_onlines.created_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		onlineQuery = onlineQuery.Where("qc_onlines.created_at <= ?", endDate+" 23:59:59")
	}

	onlineQuery.Scan(&onlineResults)

	// Add online results to details
	for _, r := range onlineResults {
		details = append(details, BoxUsageDetail{
			TrackingNumber: r.TrackingNumber,
			OrderGineeID:   r.OrderGineeID,
			BoxName:        r.BoxName,
			Quantity:       r.Quantity,
			QcBy:           r.FullName,
			CreatedAt:      r.CreatedAt.Format("02-01-2006 15:04:05"),
			Source:         "online",
		})
	}

	return details
}

// GetBoxReports generates box usage reports
// @Summary Get Box Usage Reports
// @Description Generate box usage reports with optional filters
// @Tags Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param startDate query string false "Filter by start date (YYYY-MM-DD format)"
// @Param endDate query string false "Filter by end date (YYYY-MM-DD format)"
// @Param boxName query string false "Filter term for box name"
// @Success 200 {object} utils.SuccessTotaledResponse{data=[]BoxCountReportsListResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/reports/boxes [get]
func (rc *ReportController) GetBoxReports(c fiber.Ctx) error {
	log.Println("GetBoxReports called")
	// Parse query parameters
	startDate := c.Query("startDate", "")
	endDate := c.Query("endDate", "")
	boxName := c.Query("boxName", "")

	// Build subquery for ribbon counts
	ribbonCountSubquery := rc.DB.Table("qc_ribbon_details").
		Select("qc_ribbon_details.box_id, COALESCE(SUM(qc_ribbon_details.quantity), 0) as ribbon_count").
		Joins("LEFT JOIN qc_ribbons ON qc_ribbons.id = qc_ribbon_details.qc_ribbon_id")

	// Apply date filters for ribbon
	if startDate != "" {
		ribbonCountSubquery = ribbonCountSubquery.Where("qc_ribbons.created_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		ribbonCountSubquery = ribbonCountSubquery.Where("qc_ribbons.created_at <= ?", endDate+" 23:59:59")
	}

	ribbonCountSubquery = ribbonCountSubquery.Group("qc_ribbon_details.box_id")

	// Build subquery for online counts
	onlineCountSubquery := rc.DB.Table("qc_online_details").
		Select("qc_online_details.box_id, COALESCE(SUM(qc_online_details.quantity), 0) as online_count").
		Joins("LEFT JOIN qc_onlines ON qc_onlines.id = qc_online_details.qc_online_id")

	// Apply date filters for online
	if startDate != "" {
		onlineCountSubquery = onlineCountSubquery.Where("qc_onlines.created_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		onlineCountSubquery = onlineCountSubquery.Where("qc_onlines.created_at <= ?", endDate+" 23:59:59")
	}

	onlineCountSubquery = onlineCountSubquery.Group("qc_online_details.box_id")

	// Main query with joins to subqueries
	type BoxCountResult struct {
		BoxID       uint
		BoxCode     string
		BoxName     string
		RibbonCount int
		OnlineCount int
		TotalCount  int
	}

	var results []BoxCountResult
	query := rc.DB.Table("boxes").
		Select("boxes.id as box_id, boxes.box_code, boxes.box_name, COALESCE(ribbon.ribbon_count, 0) as ribbon_count, COALESCE(online.online_count, 0) as online_count, (COALESCE(ribbon.ribbon_count, 0) + COALESCE(online.online_count, 0)) as total_count").
		Joins("LEFT JOIN (?) as ribbon ON ribbon.box_id = boxes.id", ribbonCountSubquery).
		Joins("LEFT JOIN (?) as online ON online.box_id = boxes.id", onlineCountSubquery)

	// Apply filter by box name with exact match
	if boxName != "" {
		query = query.Where("boxes.box_name = ?", boxName)
	}

	// Group by boxes columns
	query = query.Group("boxes.id, boxes.box_code, boxes.box_name, ribbon.ribbon_count, online.online_count")

	// Only show boxes with usage
	query = query.Having("(COALESCE(ribbon.ribbon_count, 0) + COALESCE(online.online_count, 0)) > 0")

	// Order by total count descending
	query = query.Order("box_id ASC")

	// Execute query
	if err := query.Scan(&results).Error; err != nil {
		log.Println("GetBoxReports - Failed to retrieve box reports:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil laporan box",
		})
	}

	// Build response with details
	reports := []BoxCountReport{}
	for _, result := range results {
		report := BoxCountReport{
			BoxID:       result.BoxID,
			BoxCode:     result.BoxCode,
			BoxName:     result.BoxName,
			TotalCount:  result.TotalCount,
			RibbonCount: result.RibbonCount,
			OnlineCount: result.OnlineCount,
		}

		// Get detailed usage for this box
		report.Details = rc.BuildBoxUsageDetails(result.BoxID, startDate, endDate)

		reports = append(reports, report)
	}

	response := BoxCountReportsListResponse{
		Reports: reports,
	}

	// Build success message with all filters
	message := "Box usage reports retrieved successfully"
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

	if boxName != "" {
		filters = append(filters, "boxName: "+boxName)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	log.Println("GetBoxReports completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessTotaledResponse{
		Success: true,
		Message: message,
		Data:    response,
		Total:   int64(len(reports)),
	})
}

// GetOutboundReports generates outbound reports
// @Summary Get Outbound Reports
// @Description Generate outbound reports with optional filters
// @Tags Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date query string false "Filter by date (YYYY-MM-DD format)"
// @Param slug query string false "Filter term for outbound slug"
// @Success 200 {object} utils.SuccessTotaledResponse{data=OutboundReportsListResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/reports/outbounds [get]
func (rc *ReportController) GetOutboundReports(c fiber.Ctx) error {
	log.Println("GetOutboundReports called")
	// Parse query parameters
	date := c.Query("date", "")
	slug := c.Query("slug", "")

	// Build base query
	var outbounds []models.Outbound
	query := rc.DB.Model(&models.Outbound{}).Preload("OutboundUser").Order("created_at DESC")

	// Apply date filters
	if date != "" {
		// Parse date dan validate format
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format tanggal tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
		// Filter for the entire day (from 00:00:00 to 23:59:59)
		startOfDay := parsedDate.Format("2006-01-02 15:04:05")
		endOfDay := parsedDate.AddDate(0, 0, 1).Format("2006-01-02 15:04:05")
		query = query.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)
	}

	// Apply filter by slug with exact match
	if slug != "" {
		query = query.Where("expedition_slug = ?", slug)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// retrieve results
	if err := query.Find(&outbounds).Error; err != nil {
		log.Println("GetOutboundReports - Failed to retrieve outbound reports:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil laporan outbound",
		})
	}

	// Format response
	outboundList := make([]models.OutboundResponse, len(outbounds))
	for i, outbound := range outbounds {
		outboundList[i] = *outbound.ToResponse()
	}

	// Build success message
	message := "Outbound reports retrieved successfully"
	var filters []string

	if date != "" {
		filters = append(filters, "date: "+date)
	}

	if slug != "" {
		filters = append(filters, "slug: "+slug)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	response := OutboundReportsListResponse{
		Outbounds: outboundList,
	}

	log.Println("GetOutboundReports completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessTotaledResponse{
		Success: true,
		Message: message,
		Data:    response,
		Total:   total,
	})
}

// GetReturnReports generates return reports
// @Summary Get Return Reports
// @Description Generate return reports with optional filters
// @Tags Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date query string false "Filter by date (YYYY-MM-DD format)"
// @Param channelId query string false "Filter term for channel ID"
// @Param storeId query string false "Filter term for store ID"
// @Success 200 {object} utils.SuccessTotaledResponse{data=[]models.ReturnResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/reports/returns [get]
func (rc *ReportController) GetReturnReports(c fiber.Ctx) error {
	log.Println("GetReturnReports called")
	// Parse query parameters
	date := c.Query("date", "")
	channelId := c.Query("channelId", "")
	storeId := c.Query("storeId", "")

	// Build base query
	var returns []models.Return
	query := rc.DB.Model(&models.Return{}).Order("created_at DESC")

	// Apply date filters
	if date != "" {
		// Parse date dan validate format
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format tanggal tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
		// Filter for the entire day (from 00:00:00 to 23:59:59)
		startOfDay := parsedDate.Format("2006-01-02 15:04:05")
		endOfDay := parsedDate.AddDate(0, 0, 1).Format("2006-01-02 15:04:05")
		query = query.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)
	}

	// Apply filter by channel ID
	if channelId != "" {
		query = query.Where("channel_id = ?", channelId)
	}

	// Apply filter by store ID
	if storeId != "" {
		query = query.Where("store_id = ?", storeId)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// retrieve results
	if err := query.Preload("ReturnDetails").Preload("Channel").Preload("Store").Preload("CreateUser").Preload("UpdateUser").Find(&returns).Error; err != nil {
		log.Println("GetReturnReports - Failed to retrieve return reports:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil laporan retur",
		})
	}

	// Format response
	returnList := make([]models.ReturnResponse, len(returns))
	for i, ret := range returns {
		returnList[i] = ret.ToResponse()
	}

	// Build success message
	message := "Return reports retrieved successfully"
	var filters []string

	if date != "" {
		filters = append(filters, "date: "+date)
	}

	if channelId != "" {
		filters = append(filters, "channelId: "+channelId)
	}

	if storeId != "" {
		filters = append(filters, "storeId: "+storeId)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	response := ReturnReportsListResponse{
		Returns: returnList,
	}

	log.Println("GetReturnReports completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessTotaledResponse{
		Success: true,
		Message: message,
		Data:    response,
		Total:   total,
	})
}

// GetComplaintReports generates complaint reports
// @Summary Get Complaint Reports
// @Description Generate complaint reports with optional filters
// @Tags Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date query string false "Filter by date (YYYY-MM-DD format)"
// @Success 200 {object} utils.SuccessTotaledResponse{data=ComplaintReportsListResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/reports/complains [get]
func (rc *ReportController) GetComplainReports(c fiber.Ctx) error {
	log.Println("GetComplainReports called")
	var complaints []models.Complain

	// Build base query
	query := rc.DB.Model(&models.Complain{}).Preload("ComplainProductDetails").Preload("ComplainUserDetails.User").Preload("Channel").Preload("Store").Preload("CreateUser").Order("created_at DESC")

	// Apply date filters if provided
	date := c.Query("date")
	if date != "" {
		// Parse date dan validate format
		if parsedDate, err := time.Parse("2006-01-02", date); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format tanggal tidak valid. Gunakan YYYY-MM-DD.",
			})
		} else {
			// Filter for the entire day (from 00:00:00 to 23:59:59)
			startOfDay := parsedDate.Format("2006-01-02 00:00:00")
			endOfDay := parsedDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
			query = query.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)
		}
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Execute query
	if err := query.Find(&complaints).Error; err != nil {
		log.Println("GetComplainReports - Failed to retrieve complaint reports:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil laporan complain",
		})
	}

	// Format response
	complaintList := make([]models.ComplainResponse, len(complaints))
	for i, complaint := range complaints {
		complaintList[i] = *complaint.ToComplainResponse()
	}

	response := ComplaintReportsListResponse{
		Complaints: complaintList,
	}

	// Build success message
	message := "Complaint reports retrieved successfully"
	var filters []string

	if date != "" {
		filters = append(filters, "date: "+date)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	log.Println("GetComplainReports completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessTotaledResponse{
		Success: true,
		Message: message,
		Data:    response,
		Total:   total,
	})
}

// GetUserFeeReports generates user fee reports
// @Summary Get User Fee Reports
// @Description Generate user fee reports with optional filters
// @Tags Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number for pagination (default is 1)"
// @Param limit query int false "Number of items per page for pagination (default is 10)"
// @Param startDate query string false "Filter by start date (YYYY-MM-DD format)"
// @Param endDate query string false "Filter by end date (YYYY-MM-DD format)"
// @Param userId query string false "Filter term for user ID"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=UserFeeReportsWithDetailsListResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/reports/user-fees [get]
func (rc *ReportController) GetUserFeeReports(c fiber.Ctx) error {
	log.Println("GetUserFeeReports called")
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	// Parse filter parameters
	userId := c.Query("userId", "")
	startDate := c.Query("startDate", "")
	endDate := c.Query("endDate", "")

	// Validate date formats
	if startDate != "" {
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format startDate tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
	}
	if endDate != "" {
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format endDate tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
	}

	// Build query to get user summaries
	type UserSummary struct {
		UserID          uint
		Username        string
		FullName        string
		Email           string
		TotalComplaints int
		TotalFeeCharge  uint
	}

	summaryQuery := rc.DB.Table("complain_user_details").
		Select("users.id as user_id, users.username, users.full_name, users.email, COUNT(DISTINCT complain_user_details.complain_id) as total_complaints, COALESCE(SUM(complain_user_details.fee_charge), 0) as total_fee_charge").
		Joins("LEFT JOIN users ON users.id = complain_user_details.user_id").
		Joins("LEFT JOIN complains ON complains.id = complain_user_details.complain_id")

	// Apply date filters on complains table
	if startDate != "" {
		summaryQuery = summaryQuery.Where("complains.updated_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		summaryQuery = summaryQuery.Where("complains.updated_at <= ?", endDate+" 23:59:59")
	}

	// Apply user filter
	if userId != "" {
		summaryQuery = summaryQuery.Where("complain_user_details.user_id = ?", userId)
	}

	summaryQuery = summaryQuery.Group("users.id, users.username, users.full_name, users.email").
		Order("total_fee_charge DESC")

	// Get total count
	var totalCount int64
	countQuery := rc.DB.Table("(?) as summaries", summaryQuery)
	if err := countQuery.Count(&totalCount).Error; err != nil {
		log.Println("GetUserFeeReports - Failed to count user fee reports:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghitung laporan biaya pengguna",
		})
	}

	// Apply pagination
	var summaries []UserSummary
	if err := summaryQuery.Limit(limit).Offset(offset).Scan(&summaries).Error; err != nil {
		log.Println("GetUserFeeReports - Failed to retrieve user fee reports:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil laporan biaya pengguna",
		})
	}

	// Build detailed reports for each user
	reports := []UserFeeReportWithDetails{}
	for _, summary := range summaries {
		// Get detailed complain information for this user
		detailQuery := rc.DB.Table("complain_user_details").
			Select("complain_user_details.complain_id, complains.code as complain_code, complains.tracking_number as tracking, complains.order_ginee_id, complain_user_details.fee_charge, complains.updated_at as complain_updated_at").
			Joins("LEFT JOIN complains ON complains.id = complain_user_details.complain_id").
			Where("complain_user_details.user_id = ?", summary.UserID)

		// Apply same date filters
		if startDate != "" {
			detailQuery = detailQuery.Where("complains.updated_at >= ?", startDate+" 00:00:00")
		}
		if endDate != "" {
			detailQuery = detailQuery.Where("complains.updated_at <= ?", endDate+" 23:59:59")
		}

		detailQuery = detailQuery.Order("complains.updated_at DESC")

		// Scan into temporary struct with time.Time
		type ComplainDetailRaw struct {
			ComplainID        uint
			ComplainCode      string
			Tracking          string
			OrderGineeID      string
			FeeCharge         uint
			ComplainUpdatedAt time.Time
		}

		var rawDetails []ComplainDetailRaw
		if err := detailQuery.Scan(&rawDetails).Error; err != nil {
			log.Println("GetUserFeeReports - Failed to retrieve complain details:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Gagal mengambil detail complain",
			})
		}

		// Format the dates
		details := []ComplainDetailInReport{}
		for _, raw := range rawDetails {
			details = append(details, ComplainDetailInReport{
				ComplainID:        raw.ComplainID,
				ComplainCode:      raw.ComplainCode,
				Tracking:          raw.Tracking,
				OrderGineeID:      raw.OrderGineeID,
				FeeCharge:         raw.FeeCharge,
				ComplainUpdatedAt: raw.ComplainUpdatedAt.Format("02-01-2006 15:04:05"),
			})
		}

		report := UserFeeReportWithDetails{
			UserID:          summary.UserID,
			Username:        summary.Username,
			FullName:        summary.FullName,
			Email:           summary.Email,
			TotalComplaints: summary.TotalComplaints,
			TotalFeeCharge:  summary.TotalFeeCharge,
			ComplainDetails: details,
		}

		reports = append(reports, report)
	}

	response := UserFeeReportsWithDetailsListResponse{
		Reports: reports,
	}

	// Build success message
	message := "User fee reports retrieved successfully"
	var filters []string

	if startDate != "" {
		filters = append(filters, "startDate: "+startDate)
	}
	if endDate != "" {
		filters = append(filters, "endDate: "+endDate)
	}
	if userId != "" {
		filters = append(filters, "userId: "+userId)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	log.Println("GetUserFeeReports completed successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    response,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: totalCount,
		},
	})
}
