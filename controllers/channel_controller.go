package controllers

import (
	"fmt"
	"livo-fiber-backend/models"
	"livo-fiber-backend/utils"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type ChannelController struct {
	DB *gorm.DB
}

func NewChannelController(db *gorm.DB) *ChannelController {
	return &ChannelController{DB: db}
}

// Request structs
type CreateChannelRequest struct {
	ChannelCode string `json:"channelCode" validate:"required,min=3,max=50"`
	ChannelName string `json:"channelName" validate:"required,min=3,max=100"`
}

type UpdateChannelRequest struct {
	ChannelCode string `json:"channelCode" validate:"required,min=3,max=50"`
	ChannelName string `json:"channelName" validate:"required,min=3,max=100"`
}

// GetChannels retrieves a list of channels with pagination and search
// @Summary Get Channels
// @Description Retrieve a list of channels with pagination and search
// @Tags Channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of channels per page" default(10)
// @Param search query string false "Search term for channel code or name"
// @Success 200 {object} utils.SuccessPaginatedResponse{data=[]models.Channel}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/channels [get]
func (bc *ChannelController) GetChannels(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var channels []models.Channel

	// Build base query
	query := bc.DB.Model(&models.Channel{}).Order("created_at DESC")

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Where("channel_code ILIKE ? OR channel_name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Limit(limit).Offset(offset).Find(&channels).Error; err != nil {
		log.Println("Error retrieving channels:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data channel",
		})
	}

	// Format response
	channelList := make([]models.ChannelResponse, len(channels))
	for i, channel := range channels {
		channelList[i] = *channel.ToResponse()
	}

	// Build success message
	message := "Channels retrieved successfully"
	var filters []string

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	// Return success response
	log.Println(message)
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    channelList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetChannel retrieves a single channel by ID
// @Summary Get Channel
// @Description Retrieve a single channel by ID
// @Tags Channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Channel}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/channels/{id} [get]
func (bc *ChannelController) GetChannel(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var channel models.Channel
	if err := bc.DB.Where("id = ?", id).First(&channel).Error; err != nil {
		log.Println("Channel with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Channel dengan id " + id + " tidak ditemukan.",
		})
	}

	log.Println("Channel retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data channel berhasil diambil",
		Data:    channel.ToResponse(),
	})
}

// CreateChannel creates a new channel
// @Summary Create Channel
// @Description Create a new channel
// @Tags Channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param channel body CreateChannelRequest true "Channel details"
// @Success 201 {object} utils.SuccessResponse{data=models.Channel}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/channels [post]
func (bc *ChannelController) CreateChannel(c fiber.Ctx) error {
	// Binding request body
	var req CreateChannelRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Convert channel code to uppercase and trim spaces
	req.ChannelCode = strings.ToUpper(strings.TrimSpace(req.ChannelCode))

	// Check for existing channel with same code
	var existingChannel models.Channel
	if err := bc.DB.Where("channel_code = ?", req.ChannelCode).First(&existingChannel).Error; err == nil {
		log.Println("Channel with code " + req.ChannelCode + " already exists.")
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Channel dengan code " + req.ChannelCode + " sudah terdaftar.",
		})
	}

	// Create new channel
	newChannel := models.Channel{
		ChannelCode: req.ChannelCode,
		ChannelName: req.ChannelName,
	}

	if err := bc.DB.Create(&newChannel).Error; err != nil {
		log.Println("Failed to create channel:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat channel",
		})
	}

	log.Println("Channel created successfully")
	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Channel berhasil dibuat",
		Data:    newChannel.ToResponse(),
	})
}

// UpdateChannel updates an existing channel by ID
// @Summary Update Channel
// @Description Update an existing channel by ID
// @Tags Channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Param request body UpdateChannelRequest true "Updated channel details"
// @Success 200 {object} utils.SuccessResponse{data=models.Channel}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/channels/{id} [put]
func (bc *ChannelController) UpdateChannel(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var channel models.Channel
	if err := bc.DB.Where("id = ?", id).First(&channel).Error; err != nil {
		log.Println("Channel with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Channel dengan id " + id + " tidak ditemukan.",
		})
	}

	// Binding request body
	var req UpdateChannelRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Convert channel code to uppercase and trim spaces
	req.ChannelCode = strings.ToUpper(strings.TrimSpace(req.ChannelCode))

	// Check for existing channel with same code (excluding current channel)
	var existingChannel models.Channel
	if err := bc.DB.Where("channel_code = ? AND id != ?", req.ChannelCode, id).First(&existingChannel).Error; err == nil {
		log.Println("Channel with code " + req.ChannelCode + " already exists.")
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Channel dengan code " + req.ChannelCode + " sudah terdaftar.",
		})
	}

	// Update channel fields
	channel.ChannelCode = req.ChannelCode
	channel.ChannelName = req.ChannelName

	if err := bc.DB.Save(&channel).Error; err != nil {
		log.Println("Failed to update channel:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui channel",
		})
	}

	log.Println("Channel updated successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Channel berhasil diperbarui",
		Data:    channel.ToResponse(),
	})
}

// DeleteChannel deletes a channel by ID
// @Summary Delete Channel
// @Description Delete a channel by ID
// @Tags Channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/channels/{id} [delete]
func (bc *ChannelController) DeleteChannel(c fiber.Ctx) error {
	// Parse id parameter
	id := c.Params("id")
	var channel models.Channel
	if err := bc.DB.Where("id = ?", id).First(&channel).Error; err != nil {
		log.Println("Channel with id " + id + " not found.")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Channel dengan id " + id + " tidak ditemukan.",
		})
	}

	// Delete channel (also deletes associated records if any due to foreign key constraints)
	if err := bc.DB.Delete(&channel).Error; err != nil {
		log.Println("Failed to delete channel:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menghapus channel",
		})
	}

	log.Println("Channel deleted successfully")
	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse{
		Success: true,
		Message: "Berhasil menghapus channel",
	})
}
