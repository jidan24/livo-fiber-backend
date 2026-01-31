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

type MobileAttendanceController struct {
	DB *gorm.DB
}

func NewMobileAttendanceController(db *gorm.DB) *MobileAttendanceController {
	return &MobileAttendanceController{DB: db}
}

// Unique response structs
type MobileCheckInResponse struct {
	Matched    bool                 `json:"matched" example:"true"`
	UserID     string               `json:"userId" example:"1"`
	Confidence float64              `json:"confidence" example:"0.95"`
	User       *models.UserResponse `json:"user"`
	Attendance *models.Attendance   `json:"attendance"`
	Status     string               `json:"status" example:"fullday"`
	Late       int                  `json:"late" example:"2"`
}

type MobileCheckOutResponse struct {
	Matched    bool                 `json:"matched" example:"true"`
	UserID     string               `json:"userId" example:"1"`
	Confidence float64              `json:"confidence" example:"0.95"`
	User       *models.UserResponse `json:"user"`
	Attendance *models.Attendance   `json:"attendance"`
	Status     string               `json:"status" example:"halfday"`
	Overtime   int                  `json:"overtime" example:"30"`
}

// VerifyUserFace verifies a user's face
// @Summary Verify User Face
// @Description Verify the logged-in user's face against their registered face
// @Tags Mobile Attendances
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param image formData file true "Face image to verify"
// @Success 200 {object} utils.SuccessResponse{data=utils.VerifyResult}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-attendances/face-verify [post]
func (mac *MobileAttendanceController) VerifyUserFace(c fiber.Ctx) error {
	log.Println("VerifyUserFace called")
	// Get current user ID from context
	currUserID := c.Locals("userId").(string)

	// Get user from database
	var user models.User
	if err := mac.DB.Where("id = ?", currUserID).First(&user).Error; err != nil {
		log.Println("VerifyUserFace - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak ditemukan",
		})
	}

	file, err := c.FormFile("image")
	if err != nil {
		log.Println("VerifyUserFace - Image file required:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "File gambar wajib diunggah",
		})
	}

	// Validate mime type
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		log.Println("VerifyUserFace - Invalid image file type")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tipe file gambar tidak valid",
		})
	}

	tmpPath := fmt.Sprintf("tmp/verify_%d.jpg", user.ID)
	if err := c.SaveFile(file, tmpPath); err != nil {
		log.Println("VerifyUserFace - Failed to save image file:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyimpan file gambar",
		})
	}
	defer os.Remove(tmpPath)

	result, err := utils.SendToDeepFaceVerify(user.ID, tmpPath)
	if err != nil {
		log.Println("VerifyUserFace - Face verification failed:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Verifikasi wajah gagal: %v", err),
		})
	}

	if !result.Matched {
		log.Printf("VerifyUserFace - Face does not match (userID=%s, confidence=%.2f)\n", currUserID, result.Confidence)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":    false,
			"error":      "Face verification failed - face does not match",
			"matched":    result.Matched,
			"userId":     result.UserID,
			"confidence": result.Confidence,
		})
	}

	// Attendance logging can be implemented here
	log.Println("VerifyUserFace completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Verifikasi wajah berhasil",
		Data:    result,
	})
}

// MobileCheckInUser checks in a user via mobile with face verification and gps location verification from location ID (limited to 10 meters accuracy form registered location)
// @Summary Mobile User Check-In
// @Description Check-in for a user via mobile
// @Tags Mobile Attendances
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param image formData file true "Face image for verification"
// @Param location_id formData int true "Location ID for GPS verification"
// @Param latitude formData float64 true "Latitude for GPS verification"
// @Param longitude formData float64 true "Longitude for GPS verification"
// @Param accuracy formData float64 true "GPS accuracy in meters"
// @Success 200 {object} utils.SuccessResponse{data=MobileCheckInResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-attendances/checkin/face [post]
func (mac *MobileAttendanceController) MobileCheckInUserByFace(c fiber.Ctx) error {
	log.Println("MobileCheckInUserByFace called")
	// Get current user ID from context
	currUserID := c.Locals("userId").(string)

	// Get user from database
	var user models.User
	if err := mac.DB.Where("id = ?", currUserID).First(&user).Error; err != nil {
		log.Println("MobileCheckInUserByFace - User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak ditemukan",
		})
	}

	file, err := c.FormFile("image")
	if err != nil {
		log.Println("MobileCheckInUserByFace - Image file required:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "File gambar wajib diunggah",
		})
	}

	// Validate mime type
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		log.Println("MobileCheckInUserByFace - Invalid image file type")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Tipe file gambar tidak valid",
		})
	}

	tmpPath := "tmp/search_face.jpg"
	if err := c.SaveFile(file, tmpPath); err != nil {
		log.Println("MobileCheckInUserByFace - Failed to save image:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyimpan file gambar",
		})
	}
	defer os.Remove(tmpPath)

	result, err := utils.SendToDeepFaceVerify(user.ID, tmpPath)
	if err != nil {
		log.Println("MobileCheckInUserByFace - Face verification failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Verifikasi wajah gagal: %v", err),
		})
	}

	if !result.Matched {
		log.Printf("MobileCheckInUserByFace - Face does not match (userID=%s)\n", currUserID)
		return c.JSON(fiber.Map{
			"matched": false,
		})
	}
	log.Println("MobileCheckInUserByFace - Face verified successfully")

	locationIDStr := c.FormValue("location_id")
	if locationIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID lokasi wajib diisi",
		})
	}

	locationID, err := strconv.Atoi(locationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID lokasi tidak valid",
		})
	}

	// Get user's current GPS coordinates
	latitudeStr := c.FormValue("latitude")
	longitudeStr := c.FormValue("longitude")
	accuracyStr := c.FormValue("accuracy")

	if latitudeStr == "" || longitudeStr == "" || accuracyStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Latitude, longitude, and akurasi wajib diisi",
		})
	}

	latitude, err := strconv.ParseFloat(latitudeStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Format latitude tidak valid",
		})
	}

	longitude, err := strconv.ParseFloat(longitudeStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Format longitude tidak valid",
		})
	}

	accuracy, err := strconv.ParseFloat(accuracyStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Format akurasi tidak valid",
		})
	}

	// Verify location exists
	var location models.Location
	if err := mac.DB.Where("id = ?", locationID).First(&location).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Lokasi tidak ditemukan",
		})
	}

	// Calculate distance between user's GPS and registered location
	distance := utils.CalculateDistance(latitude, longitude, location.Latitude, location.Longitude)

	// Check if user is within 10 meters
	if distance > 10.0 {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Anda terlalu jauh dari lokasi check-in. Jarak: %.2f meter", distance),
		})
	}

	// Fake GPS Detection
	// Check user's recent attendance records for suspicious patterns
	var recentAttendances []models.Attendance
	mac.DB.Where("user_id = ?", user.ID).
		Order("checked_in DESC").
		Limit(5).
		Find(&recentAttendances)

	// 1. Check for sudden accuracy jumps
	if len(recentAttendances) > 0 {
		lastAccuracy := recentAttendances[0].Accuracy
		accuracyDiff := accuracy - lastAccuracy
		if accuracyDiff < 0 {
			accuracyDiff = -accuracyDiff
		}

		// If accuracy suddenly jumps more than 50 meters, it's suspicious
		if accuracyDiff > 50 {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Terdeteksi perilaku GPS mencurigakan: Akurasi tiba-tiba berubah dari %.1f menjadi %.1f meter", lastAccuracy, accuracy),
			})
		}
	}

	// 2. Check for fixed accuracy (always the same value)
	if len(recentAttendances) >= 3 {
		allSame := true
		for _, att := range recentAttendances[:3] {
			if att.Accuracy != accuracy {
				allSame = false
				break
			}
		}

		if allSame && accuracy > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Terdeteksi perilaku GPS mencurigakan: Nilai akurasi terlalu konsisten",
			})
		}
	}

	// 3. Check for impossible speed
	if len(recentAttendances) > 0 {
		lastAttendance := recentAttendances[0]
		timeDiff := time.Since(lastAttendance.CheckedIn).Seconds()

		// Only check if last check-in was within the last hour
		if timeDiff < 3600 && timeDiff > 60 {
			distanceTraveled := utils.CalculateDistance(
				latitude, longitude,
				lastAttendance.Latitude, lastAttendance.Longitude,
			)

			// Calculate speed in meters per second
			speed := distanceTraveled / timeDiff

			// If speed is more than 50 m/s (180 km/h), it's suspicious
			if speed > 50 {
				return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
					Success: false,
					Error:   fmt.Sprintf("Terdeteksi perilaku GPS mencurigakan: Kecepatan perjalanan tidak mungkin (%.2f km/h)", speed*3.6),
				})
			}
		}
	}

	// 4. Check if accuracy is too poor (more than 30 meters)
	if accuracy > 30 {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Akurasi GPS terlalu rendah: %.1f meter. Pastikan GPS aktif dan coba lagi.", accuracy),
		})
	}

	// Check if user already checked in today
	var attendance models.Attendance
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	if err := mac.DB.Where("user_id = ? AND checked_in >= ? AND checked_in < ? AND checked = ?", user.ID, startOfDay, endOfDay, true).First(&attendance).Error; err == nil {
		log.Println("MobileCheckInUserByFace - User already checked in today")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna sudah melakukan check-in hari ini",
		})
	}
	log.Println("MobileCheckInUserByFace - No check-in found for today, proceeding...")

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
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Waktu check-in untuk shift penuh telah berakhir. Batas waktu %s", fulldayCheckInEnd.Format("15:04")),
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
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Waktu check-in untuk shift setengah hari telah berakhir. Batas waktu %s", halfdayCheckInEnd.Format("15:04")),
			})
		}

		if checkedInTime.After(workStartTime) {
			lateMinutes = int(checkedInTime.Sub(workStartTime).Minutes())
		}
	} else {
		// Not within any valid check-in window
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error: fmt.Sprintf("Not within valid check-in time. Fullday: %s-%s, Halfday: %s-%s",
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
		LocationID: uint(locationID),
		Latitude:   latitude,
		Longitude:  longitude,
		Accuracy:   accuracy,
	}
	log.Printf("MobileCheckInUserByFace - Creating attendance (status=%s, late=%d min)\n", status, lateMinutes)

	if err := mac.DB.Create(&newAttendance).Error; err != nil {
		log.Println("MobileCheckInUserByFace - Failed to create attendance:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal membuat data absensi",
		})
	}

	// Reload with associations
	mac.DB.Preload("User").Preload("Location").First(&newAttendance, newAttendance.ID)

	log.Println("MobileCheckInUserByFace completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil melakukan check-in",
		Data: MobileCheckInResponse{
			Matched:    true,
			UserID:     strconv.Itoa(int(user.ID)),
			Confidence: result.Confidence,
			User:       user.ToResponse(),
			Attendance: &newAttendance,
			Status:     status,
			Late:       lateMinutes,
		},
	})
}

// MobileCheckOutUser checks out a user via mobile with face verification and gps location verification from location ID (limited to 10 meters accuracy form registered location)
// @Summary Mobile User Check-Out
// @Description Check-out for a user via mobile
// @Tags Mobile Attendances
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param image formData file true "Face image for verification"
// @Param location_id formData int true "Location ID for GPS verification"
// @Param latitude formData float64 true "Latitude for GPS verification"
// @Param longitude formData float64 true "Longitude for GPS verification"
// @Param accuracy formData float64 true "GPS accuracy in meters"
// @Success 200 {object} utils.SuccessResponse{data=MobileCheckOutResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/mobile-attendances/checkout/face [put]
func (mac *MobileAttendanceController) MobileCheckOutUserByFace(c fiber.Ctx) error {
	// Get current user ID from context
	currUserID := c.Locals("userId").(string)

	// Get user from database
	var user models.User
	if err := mac.DB.Where("id = ?", currUserID).First(&user).Error; err != nil {
		log.Println("User not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna tidak ditemukan",
		})
	}

	file, err := c.FormFile("image")
	if err != nil {
		log.Println("Image file required:", err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "File gambar wajib diubah",
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
		log.Println("Failed to save image:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal menyimpan file gambar",
		})
	}
	defer os.Remove(tmpPath)

	result, err := utils.SendToDeepFaceVerify(user.ID, tmpPath)
	if err != nil {
		log.Println("Face verification failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Gagal verifikasi wajah: %v", err),
		})
	}

	if !result.Matched {
		log.Printf("Face does not match (userID=%s)\n", currUserID)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Verifikasi wajah gagal-wajah tidak cocok",
		})
	}

	locationIDStr := c.FormValue("location_id")
	if locationIDStr == "" {
		log.Println("Location ID is required")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID lokasi wajib diisi",
		})
	}

	locationID, err := strconv.Atoi(locationIDStr)
	if err != nil {
		log.Println("Invalid Location ID")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "ID lokasi tidak valid",
		})
	}

	// Get user's current GPS coordinates
	latitudeStr := c.FormValue("latitude")
	longitudeStr := c.FormValue("longitude")
	accuracyStr := c.FormValue("accuracy")

	if latitudeStr == "" || longitudeStr == "" || accuracyStr == "" {
		log.Println("Latitude, longitude, and accuracy are required")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Latitude, longitude, and akurasi wajib diisi",
		})
	}

	latitude, err := strconv.ParseFloat(latitudeStr, 64)
	if err != nil {
		log.Println("Invalid latitude format")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "format latitude tidak valid",
		})
	}

	longitude, err := strconv.ParseFloat(longitudeStr, 64)
	if err != nil {
		log.Println("Invalid longitude format")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "format longitude tidak valid",
		})
	}

	accuracy, err := strconv.ParseFloat(accuracyStr, 64)
	if err != nil {
		log.Println("Invalid accuracy format")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "format accuracy tidak valid",
		})
	}

	// Verify location exists
	var location models.Location
	if err := mac.DB.Where("id = ?", locationID).First(&location).Error; err != nil {
		log.Println("Location not found")
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Lokasi tidak ditemukan",
		})
	}

	// Calculate distance between user's GPS and registered location
	distance := utils.CalculateDistance(latitude, longitude, location.Latitude, location.Longitude)

	// Check if user is within 10 meters
	if distance > 10.0 {
		log.Println("User is too far from the check-in location")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Anda terlalu jauh dari lokasi check-in. Jarak: %.2f meter", distance),
		})
	}

	// Fake GPS Detection
	// Check user's recent attendance records for suspicious patterns
	var recentAttendances []models.Attendance
	mac.DB.Where("user_id = ?", user.ID).
		Order("checked_in DESC").
		Limit(5).
		Find(&recentAttendances)

	// 1. Check for sudden accuracy jumps
	if len(recentAttendances) > 0 {
		lastAccuracy := recentAttendances[0].Accuracy
		accuracyDiff := accuracy - lastAccuracy
		if accuracyDiff < 0 {
			accuracyDiff = -accuracyDiff
		}

		// If accuracy suddenly jumps more than 50 meters, it's suspicious
		if accuracyDiff > 50 {
			log.Println("Suspicious GPS behavior detected")
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Terdeteksi perilaku GPS mencurigakan: Akurasi tiba-tiba berubah dari %.1f Menjadi %.1f meter", lastAccuracy, accuracy),
			})
		}
	}

	// 2. Check for fixed accuracy (always the same value)
	if len(recentAttendances) >= 3 {
		allSame := true
		for _, att := range recentAttendances[:3] {
			if att.Accuracy != accuracy {
				allSame = false
				break
			}
		}

		if allSame && accuracy > 0 {
			log.Println("Suspicious GPS behavior detected: Accuracy values are suspiciously consistent")
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Success: false,
				Error:   "Terdeteksi perilaku GPS mencurigakan: Nilai akurasi terlalu konsisten",
			})
		}
	}

	// 3. Check for impossible speed
	if len(recentAttendances) > 0 {
		lastAttendance := recentAttendances[0]
		timeDiff := time.Since(lastAttendance.CheckedIn).Seconds()

		// Only check if last check-in was within the last hour
		if timeDiff < 3600 && timeDiff > 60 {
			distanceTraveled := utils.CalculateDistance(
				latitude, longitude,
				lastAttendance.Latitude, lastAttendance.Longitude,
			)

			// Calculate speed in meters per second
			speed := distanceTraveled / timeDiff

			// If speed is more than 50 m/s (180 km/h), it's suspicious
			if speed > 50 {
				log.Println("Suspicious GPS behavior detected: Impossible travel speed")
				return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
					Success: false,
					Error:   fmt.Sprintf("Terdeteksi perilaku GPS mencurigakan: Kecepatan perjalanan tidak mungkin (%.2f km/h)", speed*3.6),
				})
			}
		}
	}

	// 4. Check if accuracy is too poor (more than 30 meters)
	if accuracy > 30 {
		log.Println("GPS accuracy is too poor")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("Akurasi GPS terlalu rendah: %.1f meter. Pastikan GPS aktif dan coba lagi.", accuracy),
		})
	}

	// Find today's attendance record
	var attendance models.Attendance
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	if err := mac.DB.Where("user_id = ? AND checked_in >= ? AND checked_in < ? AND checked = ?", user.ID, startOfDay, endOfDay, true).First(&attendance).Error; err != nil {
		log.Println("User has not checked in today")
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Pengguna belum melakukan check-in hari ini",
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
		log.Printf("Not within valid check-out time. Early checkout: %s-%s, Regular checkout: %s onwards",
			earlyCheckOut.Format("15:04"), earlyCheckOutEnd.Format("15:04"),
			regularCheckOutStart.Format("15:04"))
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Success: false,
			Error: fmt.Sprintf("Tidak berada dalam waktu check-out yang valid. Cehck-out awal: %s-%s, Check-out reguler: %s Mulai",
				earlyCheckOut.Format("15:04"), earlyCheckOutEnd.Format("15:04"),
				regularCheckOutStart.Format("15:04")),
		})
	}

	// Update attendance record
	if err := mac.DB.Save(&attendance).Error; err != nil {
		log.Println("Failed to update attendance record:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Success: false,
			Error:   "Gagal memperbarui data absensi",
		})
	}

	log.Println("MobileCheckOutUserByFace completed successfully")
	return c.JSON(utils.SuccessResponse{
		Success: true,
		Message: "Pengguna berhasil melakukan check-out",
		Data: MobileCheckOutResponse{
			Matched:    true,
			UserID:     result.UserID,
			Confidence: result.Confidence,
			User:       user.ToResponse(),
			Attendance: &attendance,
			Status:     attendance.Status,
			Overtime:   overtime,
		},
	})
}
