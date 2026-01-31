package controllers

import (
	"fmt"
	"livo-fiber-backend/models"
	"livo-fiber-backend/utils"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type AttendanceController struct {
	DB *gorm.DB
}

func NewAttendanceController(db *gorm.DB) *AttendanceController {
	return &AttendanceController{DB: db}

}

// Request structs
type CheckInManualRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type CheckOutManualRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Unique response structs
type CheckInResponse struct {
	Matched    bool                       `json:"matched" example:"true"`
	UserID     string                     `json:"userId" example:"1"`
	Confidence float64                    `json:"confidence" example:"0.95"`
	User       *models.UserResponse       `json:"user"`
	Attendance *models.AttendanceResponse `json:"attendance"`
	Status     string                     `json:"status" example:"fullday"`
	Late       int                        `json:"late" example:"2"`
}

type CheckInManualResponse struct {
	Matched    bool                       `json:"matched" example:"true"`
	User       *models.UserResponse       `json:"user"`
	Attendance *models.AttendanceResponse `json:"attendance"`
	Status     string                     `json:"status" example:"fullday"`
	Late       int                        `json:"late" example:"2"`
}

type CheckOutResponse struct {
	Matched    bool                       `json:"matched" example:"true"`
	UserID     string                     `json:"userId" example:"1"`
	Confidence float64                    `json:"confidence" example:"0.95"`
	User       *models.UserResponse       `json:"user"`
	Attendance *models.AttendanceResponse `json:"attendance"`
	Status     string                     `json:"status" example:"halfday"`
	Overtime   int                        `json:"overtime" example:"30"`
}

type CheckOutManualResponse struct {
	Matched    bool                       `json:"matched" example:"true"`
	User       *models.UserResponse       `json:"user"`
	Attendance *models.AttendanceResponse `json:"attendance"`
	Status     string                     `json:"status" example:"halfday"`
	Overtime   int                        `json:"overtime" example:"30"`
}

// SearchUsersByFace searches for users by face image
// @Summary Search Users by Face
// @Description Search for users by face image
// @Tags Attendances
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Face image to search for"
// @Success 200 {object} utils.SuccessResponse{data=models.UserResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/attendances/search/face [post]
func (ac *AttendanceController) SearchUsersByFace(c fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		log.Println("Image file is required")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "File gambar diperlukan",
		})
	}

	// Validate mime type
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		log.Println("Invalid image file type")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tipe file gambar tidak valid",
		})
	}

	tmpPath := "tmp/search_face.jpg"
	if err := c.SaveFile(file, tmpPath); err != nil {
		log.Println("Failed to save image file:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyimpan file gambar",
		})
	}
	defer os.Remove(tmpPath)

	result, err := utils.SendToDeepFaceSearch(tmpPath)
	if err != nil {
		log.Println("Face search failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Pencarian wajah gagal: %v", err),
		})
	}

	if !result.Matched {
		log.Println("Face not recognized")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Wajah tidak dikenali",
		})
	}

	// Fetch user data from database
	var user models.User
	if err := ac.DB.Preload("Roles").Where("id = ?", result.UserID).First(&user).Error; err != nil {
		log.Println("User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak ditemukan",
		})
	}

	log.Println("Face recognized for user ID:", result.UserID)
	return c.JSON(fiber.Map{
		"matched":    true,
		"userId":     result.UserID,
		"confidence": result.Confidence,
		"user":       user.ToResponse(),
	})
}

// CheckInUserByFace checks in a user by face image
// @Summary Check In Users by Face
// @Description Check In for users by face image
// @Tags Attendances
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Face image to search for"
// @Success 200 {object} utils.SuccessResponse{data=CheckInResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/attendances/checkin/face [post]
func (ac *AttendanceController) CheckInUserByFace(c fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		log.Println("Image file is required")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "File gambar diperlukan",
		})
	}

	// Validate mime type
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		log.Println("Invalid image file type")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tipe file gambar tidak valid",
		})
	}

	tmpPath := "tmp/search_face.jpg"
	if err := c.SaveFile(file, tmpPath); err != nil {
		log.Println("Failed to save image file:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyimpan gambar",
		})
	}
	defer os.Remove(tmpPath)

	result, err := utils.SendToDeepFaceSearch(tmpPath)
	if err != nil {
		log.Println("Face search failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Pencarian wajah gagal: %v", err),
		})
	}

	if !result.Matched {
		log.Println("Face not recognized")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Wajah tidak dikenali",
		})
	}

	// Fetch user data from database
	var user models.User
	if err := ac.DB.Preload("Roles").Where("id = ?", result.UserID).First(&user).Error; err != nil {
		log.Println("User not found")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak ditemukan",
		})
	}

	// Check if user already checked in today
	var attendance models.Attendance
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	if err := ac.DB.Where("user_id = ? AND checked_in >= ? AND checked_in < ? AND checked = ?", user.ID, startOfDay, endOfDay, true).First(&attendance).Error; err == nil {
		log.Println("User already checked in today")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna sudah melakukan check-in hari ini",
		})
	}

	// Automatically determine status based on check-in time
	checkedInTime := time.Now()

	// Define time windows for fullday and halfday
	fulldayCheckInStart := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
	fulldayCheckInEnd := time.Date(now.Year(), now.Month(), now.Day(), 8, 5, 0, 0, now.Location())
	fulldayWorkStart := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())

	halfdayCheckInStart := time.Date(now.Year(), now.Month(), now.Day(), 11, 30, 0, 0, now.Location())
	halfdayCheckInEnd := time.Date(now.Year(), now.Month(), now.Day(), 12, 35, 0, 0, now.Location())
	halfdayWorkStart := time.Date(now.Year(), now.Month(), now.Day(), 12, 30, 0, 0, now.Location())

	var status string
	var workStartTime time.Time
	var lateMinutes int

	// Check which time window the check-in falls into
	if checkedInTime.After(fulldayCheckInStart.Add(-1*time.Minute)) && checkedInTime.Before(fulldayCheckInEnd.Add(1*time.Minute)) {
		// Within fullday window (7:00 - 8:05)
		status = "fullday"
		workStartTime = fulldayWorkStart

		if checkedInTime.After(fulldayCheckInEnd) {
			log.Println("Check-in time has expired for fullday shift. Deadline was", fulldayCheckInEnd.Format("15:04"))
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Waktu check-in untuk shift seharian penuh telah berakhir. Batas waktunya adalah %s", fulldayCheckInEnd.Format("15:04")),
			})
		}

		if checkedInTime.After(workStartTime) {
			lateMinutes = int(checkedInTime.Sub(workStartTime).Minutes())
		}
	} else if checkedInTime.After(halfdayCheckInStart.Add(-1*time.Minute)) && checkedInTime.Before(halfdayCheckInEnd.Add(1*time.Minute)) {
		// Within halfday window (11:30 - 12:35)
		status = "halfday"
		workStartTime = halfdayWorkStart

		if checkedInTime.After(halfdayCheckInEnd) {
			log.Println("Check-in time has expired for halfday shift. Deadline was", halfdayCheckInEnd.Format("15:04"))
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Waktu check-in untuk shift setengah hari telah berakhir. Batas waktunya adalah %s", halfdayCheckInEnd.Format("15:04")),
			})
		}

		if checkedInTime.After(workStartTime) {
			lateMinutes = int(checkedInTime.Sub(workStartTime).Minutes())
		}
	} else {
		// Not within any valid check-in window
		log.Println("Not within valid check-in time. Fullday:", fulldayCheckInStart.Format("15:04"), "-", fulldayCheckInEnd.Format("15:04"),
			"Halfday:", halfdayCheckInStart.Format("15:04"), "-", halfdayCheckInEnd.Format("15:04"))
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error: fmt.Sprintf("Tidak dalam waktu check-in yang valid. Seharian penuh: %s-%s, Setengah hari: %s-%s",
				fulldayCheckInStart.Format("15:04"), fulldayCheckInEnd.Format("15:04"),
				halfdayCheckInStart.Format("15:04"), halfdayCheckInEnd.Format("15:04")),
		})
	}

	// Create attendance record
	newAttendance := models.Attendance{
		UserID:     user.ID,
		CheckedIn:  checkedInTime,
		Checked:    true,
		Status:     status,
		Late:       lateMinutes,
		LocationID: 1,
		Latitude:   -7.9484807,
		Longitude:  112.6460763,
		Accuracy:   1.0,
	}

	if err := ac.DB.Create(&newAttendance).Error; err != nil {
		log.Println("Failed to create attendance record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat catatan kehadiran",
		})
	}

	// Reload attendace data and related data
	ac.DB.Preload("User").Preload("Location").Where("id = ?", newAttendance.ID).First(&newAttendance)

	log.Println("User checked in successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil check-in",
		Data: CheckInResponse{
			Matched:    true,
			UserID:     result.UserID,
			Confidence: result.Confidence,
			User:       user.ToResponse(),
			Attendance: newAttendance.ToResponse(),
			Status:     status,
			Late:       lateMinutes,
		},
	})
}

// CheckOutUserByFace checks out a user by face image
// @Summary Check Out Users by Face
// @Description Check Out for users by face image
// @Tags Attendances
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Face image to search for"
// @Success 200 {object} utils.SuccessResponse{data=CheckOutResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/attendances/checkout/face [put]
func (ac *AttendanceController) CheckOutUserByFace(c fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		log.Println("Image file is required")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "File gambar diperlukan",
		})
	}

	// Validate mime type
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		log.Println("Invalid image file type")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tipe file gambar tidak valid",
		})
	}

	tmpPath := "tmp/search_face.jpg"
	if err := c.SaveFile(file, tmpPath); err != nil {
		log.Println("Failed to save image file:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyimpan file gambar",
		})
	}
	defer os.Remove(tmpPath)

	result, err := utils.SendToDeepFaceSearch(tmpPath)
	if err != nil {
		log.Println("Face search failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Pencarian wajah gagal: %v", err),
		})
	}

	if !result.Matched {
		log.Println("Face not recognized")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Wajah tidak dikenali",
		})
	}

	// Fetch user data from database
	var user models.User
	if err := ac.DB.Preload("Roles").Where("id = ?", result.UserID).First(&user).Error; err != nil {
		log.Println("User not found")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak ditemukan",
		})
	}

	// Search the target user's attendance record
	var attendance models.Attendance
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	if err := ac.DB.Where("user_id = ? AND checked_in >= ? AND checked_in < ? AND checked = ?", user.ID, startOfDay, endOfDay, true).First(&attendance).Error; err != nil {
		log.Println("Attendance record not found or user has not checked in today")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Catatan kehadiran tidak ditemukan atau pengguna belum melakukan check-in hari ini",
		})
	}

	// Automatically determine checkout behavior based on time
	checkedOutTime := time.Now()

	// Define checkout time windows
	earlyCheckOut := time.Date(now.Year(), now.Month(), now.Day(), 12, 30, 0, 0, now.Location())
	earlyCheckOutEnd := earlyCheckOut.Add(5 * time.Minute)

	regularCheckOut := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, now.Location())
	regularCheckOutStart := regularCheckOut.Add(-5 * time.Minute) // Allow 5 minutes before

	overtime := 0

	// Check if checking out around 12:30 (early checkout)
	if checkedOutTime.After(earlyCheckOut.Add(-1*time.Minute)) && checkedOutTime.Before(earlyCheckOutEnd.Add(1*time.Minute)) {
		// Update status from fullday to halfday, no overtime
		attendance.Status = "halfday"
		attendance.CheckedOut = &checkedOutTime
		attendance.Checked = false
		attendance.Overtime = 0
	} else if checkedOutTime.After(regularCheckOutStart) {
		// Checking out around 17:00 or later
		switch attendance.Status {
		case "halfday":
			// Halfday status: just update checkout time, no overtime
			attendance.CheckedOut = &checkedOutTime
			attendance.Checked = false
			attendance.Overtime = 0
		case "fullday":
			// Fullday status: update checkout and calculate overtime if after 17:00
			attendance.CheckedOut = &checkedOutTime
			attendance.Checked = false

			if checkedOutTime.After(regularCheckOut) {
				overtime = int(checkedOutTime.Sub(regularCheckOut).Minutes())
			}
			attendance.Overtime = overtime
		}
	} else {
		// Not within valid checkout windows
		log.Println("Not within valid check-out time. Early checkout:", earlyCheckOut.Format("15:04"), "-", earlyCheckOutEnd.Format("15:04"),
			"Regular checkout:", regularCheckOutStart.Format("15:04"), "onwards")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error: fmt.Sprintf("Tidak dalam jangka waktu check-out yang valid. Pembayaran lebih awal: %s-%s, Pembayaran regular: %s onwards",
				earlyCheckOut.Format("15:04"), earlyCheckOutEnd.Format("15:04"),
				regularCheckOutStart.Format("15:04")),
		})
	}

	// Update attendance record
	if err := ac.DB.Save(&attendance).Error; err != nil {
		log.Println("Failed to update attendance record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui catatan kehadiran.",
		})
	}

	// Reload attendace data and related data
	ac.DB.Preload("User").Preload("Location").Where("id = ?", attendance.ID).First(&attendance)

	log.Println("User checked out successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil check out",
		Data: CheckOutResponse{
			Matched:    true,
			UserID:     result.UserID,
			Confidence: result.Confidence,
			User:       user.ToResponse(),
			Attendance: attendance.ToResponse(),
			Status:     attendance.Status,
			Overtime:   overtime,
		},
	})
}

// CheckInUserManual allows manual check-in for a user by username and password
// @Summary Manual Check-In User
// @Description Allow manual check-in for a user by username and password
// @Tags Attendances
// @Accept json
// @Produce json
// @Param body body CheckInManualRequest true "Manual Check-In Request Body"
// @Success 200 {object} utils.SuccessResponse{data=CheckInManualResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/attendances/checkin/manual [post]
func (ac *AttendanceController) CheckInUserManual(c fiber.Ctx) error {
	// Binding request body
	var req CheckInManualRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Find user by username
	var user models.User
	if err := ac.DB.Preload("Roles").Where("username = ?", req.Username).First(&user).Error; err != nil {
		log.Println("User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak ditemukan",
		})
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		log.Println("Invalid password")
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Kata sandi tidak valid",
		})
	}

	// Proceed with check-in logic (similar to face check-in)
	// Check if user already checked in today
	var attendance models.Attendance
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	if err := ac.DB.Where("user_id = ? AND checked_in >= ? AND checked_in < ? AND checked = ?", user.ID, startOfDay, endOfDay, true).First(&attendance).Error; err == nil {
		log.Println("User already checked in today")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna sudah melakukan check-in hari ini.",
		})
	}

	// Automatically determine status based on check-in time
	checkedInTime := time.Now()

	// Define time windows for fullday and halfday
	fulldayCheckInStart := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
	fulldayCheckInEnd := time.Date(now.Year(), now.Month(), now.Day(), 8, 5, 0, 0, now.Location())
	fulldayWorkStart := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())

	halfdayCheckInStart := time.Date(now.Year(), now.Month(), now.Day(), 11, 30, 0, 0, now.Location())
	halfdayCheckInEnd := time.Date(now.Year(), now.Month(), now.Day(), 12, 35, 0, 0, now.Location())
	halfdayWorkStart := time.Date(now.Year(), now.Month(), now.Day(), 12, 30, 0, 0, now.Location())

	var status string
	var workStartTime time.Time
	var lateMinutes int

	// Check which time window the check-in falls into
	if checkedInTime.After(fulldayCheckInStart.Add(-1*time.Minute)) && checkedInTime.Before(fulldayCheckInEnd.Add(1*time.Minute)) {
		// Within fullday window (7:00 - 8:05)
		status = "fullday"
		workStartTime = fulldayWorkStart

		if checkedInTime.After(fulldayCheckInEnd) {
			log.Println("Check-in time has expired for fullday shift. Deadline was", fulldayCheckInEnd.Format("15:04"))
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Waktu check-in untuk shift seharian penuh telah berakhir. Batas waktunya adalah %s", fulldayCheckInEnd.Format("15:04")),
			})
		}

		if checkedInTime.After(workStartTime) {
			lateMinutes = int(checkedInTime.Sub(workStartTime).Minutes())
		}
	} else if checkedInTime.After(halfdayCheckInStart.Add(-1*time.Minute)) && checkedInTime.Before(halfdayCheckInEnd.Add(1*time.Minute)) {
		// Within halfday window (11:30 - 12:35)
		status = "halfday"
		workStartTime = halfdayWorkStart

		if checkedInTime.After(halfdayCheckInEnd) {
			log.Println("Check-in time has expired for halfday shift. Deadline was", halfdayCheckInEnd.Format("15:04"))
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Waktunya check-in untuk shift setengah hari telah berakhir. Batas waktunya adalah %s", halfdayCheckInEnd.Format("15:04")),
			})
		}

		if checkedInTime.After(workStartTime) {
			lateMinutes = int(checkedInTime.Sub(workStartTime).Minutes())
		}
	} else {
		// Not within any valid check-in window
		log.Println("Not within valid check-in time. Fullday:", fulldayCheckInStart.Format("15:04"), "-", fulldayCheckInEnd.Format("15:04"),
			"Halfday:", halfdayCheckInStart.Format("15:04"), "-", halfdayCheckInEnd.Format("15:04"))
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error: fmt.Sprintf("Tidak dalam waktu check-in yang valid. Sehari penuh: %s-%s, Setengah hari: %s-%s",
				fulldayCheckInStart.Format("15:04"), fulldayCheckInEnd.Format("15:04"),
				halfdayCheckInStart.Format("15:04"), halfdayCheckInEnd.Format("15:04")),
		})
	}

	// Create attendance record
	newAttendance := models.Attendance{
		UserID:     user.ID,
		CheckedIn:  checkedInTime,
		Checked:    true,
		Status:     status,
		Late:       lateMinutes,
		LocationID: 1,
		Latitude:   -7.9484807,
		Longitude:  112.6460763,
		Accuracy:   1.0,
	}

	if err := ac.DB.Create(&newAttendance).Error; err != nil {
		log.Println("Failed to create attendance record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat catatan kehadiran",
		})
	}

	// Reload attendace data and related data
	ac.DB.Preload("User").Preload("Location").Where("id = ?", newAttendance.ID).First(&newAttendance)

	log.Println("User checked in successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil check-in",
		Data: CheckInManualResponse{
			Matched:    true,
			User:       user.ToResponse(),
			Attendance: newAttendance.ToResponse(),
			Status:     status,
			Late:       lateMinutes,
		},
	})
}

// CheckOutUserManual allows manual check-out for a user by username and password
// @Summary Manual Check-Out User
// @Description Allow manual check-out for a user by username and password
// @Tags Attendances
// @Accept json
// @Produce json
// @Param body body CheckOutManualRequest true "Manual Check-Out Request Body"
// @Success 200 {object} utils.SuccessResponse{data=CheckOutManualResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/attendances/checkout/manual [put]
func (ac *AttendanceController) CheckOutUserManual(c fiber.Ctx) error {
	// Binding request body
	var req CheckOutManualRequest
	if err := c.Bind().JSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Isi permintaan tidak valid",
		})
	}

	// Find user by username
	var user models.User
	if err := ac.DB.Preload("Roles").Where("username = ?", req.Username).First(&user).Error; err != nil {
		log.Println("User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak ditemukan",
		})
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		log.Println("Invalid password")
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Kata sandi tidak valid",
		})
	}

	// Proceed with check-out logic (similar to face check-out)
	// Search the target user's attendance record
	var attendance models.Attendance
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	if err := ac.DB.Where("user_id = ? AND checked_in >= ? AND checked_in < ? AND checked = ?", user.ID, startOfDay, endOfDay, true).First(&attendance).Error; err != nil {
		log.Println("Attendance record not found or user has not checked in today")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Catatan kehadiran tidak ditemukan atau pengguna belum melakukan check-in hari ini",
		})
	}

	// Automatically determine checkout behavior based on time
	checkedOutTime := time.Now()

	// Define checkout time windows
	earlyCheckOut := time.Date(now.Year(), now.Month(), now.Day(), 12, 30, 0, 0, now.Location())
	earlyCheckOutEnd := earlyCheckOut.Add(5 * time.Minute)

	regularCheckOut := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, now.Location())
	regularCheckOutStart := regularCheckOut.Add(-5 * time.Minute) // Allow 5 minutes before

	overtime := 0

	// Check if checking out around 12:30 (early checkout)
	if checkedOutTime.After(earlyCheckOut.Add(-1*time.Minute)) && checkedOutTime.Before(earlyCheckOutEnd.Add(1*time.Minute)) {
		// Update status from fullday to halfday, no overtime
		attendance.Status = "halfday"
		attendance.CheckedOut = &checkedOutTime
		attendance.Checked = false
		attendance.Overtime = 0
	} else if checkedOutTime.After(regularCheckOutStart) {
		// Checking out around 17:00 or later
		switch attendance.Status {
		case "halfday":
			// Halfday status: just update checkout time, no overtime
			attendance.CheckedOut = &checkedOutTime
			attendance.Checked = false
			attendance.Overtime = 0
		case "fullday":
			// Fullday status: update checkout and calculate overtime if after 17:00
			attendance.CheckedOut = &checkedOutTime
			attendance.Checked = false

			if checkedOutTime.After(regularCheckOut) {
				overtime = int(checkedOutTime.Sub(regularCheckOut).Minutes())
			}
			attendance.Overtime = overtime
		}
	} else {
		// Not within valid checkout windows
		log.Println("Not within valid check-out time. Early checkout:", earlyCheckOut.Format("15:04"), "-", earlyCheckOutEnd.Format("15:04"),
			"Regular checkout:", regularCheckOutStart.Format("15:04"), "onwards")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error: fmt.Sprintf("Tidak berada dalam waktu check-out yang valid. Check-out lebih awal: %s-%s, Check-out regular: %s Mulai pukul",
				earlyCheckOut.Format("15:04"), earlyCheckOutEnd.Format("15:04"),
				regularCheckOutStart.Format("15:04")),
		})
	}

	// Update attendance record
	if err := ac.DB.Save(&attendance).Error; err != nil {
		log.Println("Failed to update attendance record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pembaruan data kehadiran gagal",
		})
	}

	// Reload attendace data and related data
	ac.DB.Preload("User").Preload("Location").Where("id = ?", attendance.ID).First(&attendance)

	log.Println("User checked out successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Check-out pengguna berhasil",
		Data: CheckOutManualResponse{
			Matched:    true,
			User:       user.ToResponse(),
			Attendance: attendance.ToResponse(),
			Status:     attendance.Status,
			Overtime:   overtime,
		},
	})
}

// GetAttendances retrieves all attendance records
// @Summary Get All Attendances
// @Description Retrieve all attendance records with pagination and search
// @Tags Attendances
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of records per page" default(10)
// @Param startDate query string false "Start date (YYYY-MM-DD format)"
// @Param endDate query string false "End date (YYYY-MM-DD format)"
// @Param search query string false "Search term for user name or username"
// @Success 200 {object} utils.SuccessResponse{data=[]models.AttendanceResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/attendances [get]
func (ac *AttendanceController) GetAttendances(c fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var attendances []models.Attendance

	// Build base query
	query := ac.DB.Model(&models.Attendance{}).Preload("User").Preload("Location").Order("checked_in DESC")

	// Date range filter if provided
	startDate := c.Query("startDate", "")
	endDate := c.Query("endDate", "")
	if startDate != "" {
		// Parse start date and set time to beginning of the day
		parsedStartDate, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			log.Println("Invalid start_date format. Use YYYY-MM-DD.")
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format start_date tidak sesuai dengan ketentuan. Gunakan format YYYY-MM-DD.",
			})
		}
		startOfDay := time.Date(parsedStartDate.Year(), parsedStartDate.Month(), parsedStartDate.Day(), 0, 0, 0, 0, parsedStartDate.Location())
		query = query.Where("checked_in >= ?", startOfDay)
	}
	if endDate != "" {
		// Parse end date and set time to end of the day
		parsedEndDate, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			log.Println("Invalid end_date format. Use YYYY-MM-DD.")
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Format end_date tidak valid. Gunakan YYYY-MM-DD.",
			})
		}
		endOfDay := time.Date(parsedEndDate.Year(), parsedEndDate.Month(), parsedEndDate.Day(), 23, 59, 59, 0, parsedEndDate.Location())
		query = query.Where("checked_in <= ?", endOfDay)
	}

	// Search condition if provided
	search := strings.TrimSpace(c.Query("search", ""))
	if search != "" {
		query = query.Joins("JOIN users ON users.id = attendances.user_id").
			Where("users.username ILIKE ? OR users.full_name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Retrieve paginated results
	if err := query.Offset(offset).Limit(limit).Find(&attendances).Error; err != nil {
		log.Println("Failed to retrieve attendances:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal mengambil data kehadiran",
		})
	}

	// Format response
	attendanceList := make([]models.AttendanceResponse, len(attendances))
	for i, attendance := range attendances {
		attendanceList[i] = *attendance.ToResponse()
	}

	// Build success message
	message := "Attendances retrieved successfully"
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

	log.Panicln(message)
	return c.Status(fiber.StatusOK).JSON(utils.SuccessPaginatedResponse{
		Success: true,
		Message: message,
		Data:    attendanceList,
		Pagination: utils.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetAttendanceByID retrieves a specific attendance record by ID
// @Summary Get Attendance by ID
// @Description Retrieve a specific attendance record by its ID
// @Tags Attendances
// @Produce json
// @Security BearerAuth
// @Param id path int true "Attendance ID"
// @Success 200 {object} utils.SuccessResponse{data=models.AttendanceResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/attendances/{id} [get]
func (ac *AttendanceController) GetAttendanceByID(c fiber.Ctx) error {
	// Parse id paramameter
	id := c.Params("id")
	var attendance models.Attendance
	if err := ac.DB.Preload("User").Preload("Location").First(&attendance, id).Error; err != nil {
		log.Println("Attendance record not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Data kehadiran tidak ditemukan",
		})
	}

	log.Println("Attendance record retrieved successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Data kehadiran berhasil diambil",
		Data:    attendance.ToResponse(),
	})
}
