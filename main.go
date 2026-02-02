package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"time"

	"livo-fiber-backend/config"
	"livo-fiber-backend/database"
	_ "livo-fiber-backend/docs" // Import generated docs
	"livo-fiber-backend/routes"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/joho/godotenv"
)

// @title Livotech Warehouse Management System API Documentation
// @version 1.0
// @description This is the API documentation for Livotech Warehouse Management System API Documentation
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@livo.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host 192.168.31.147:8040
// @BasePath /
// @schemes http https

// matchOriginPattern checks if an origin matches a pattern with wildcards
func matchOriginPattern(pattern, origin string) bool {
	// Convert pattern to regex-like matching
	// Replace * with a regex pattern that matches any characters except colon and slash
	if !strings.Contains(pattern, "*") {
		return false
	}

	// Split by wildcard
	parts := strings.Split(pattern, "*")
	if len(parts) != 2 {
		return false
	}

	// Check if origin starts with the part before * and ends with the part after *
	return strings.HasPrefix(origin, parts[0]) && strings.HasSuffix(origin, parts[1])
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	database.ConnectDatabase(cfg)
	database.MigrateDatabase()
	database.SeedInitialRole()
	database.SeedInitialBox()
	database.SeedInitialChannel()
	database.SeedInitialExpedition()
	database.SeedInitialStore()
	database.SeedInitialUser()
	database.SeedInitialLocation()

	// Get database instance
	database.GetDB()

	// Create or open log file
	logFile, err := os.OpenFile("./log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer logFile.Close()

	// Set standard log output to write to both console and file
	log.SetOutput(logFile)

	// Create Fiber app with go-joson
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
		AppName:      "Livotech Warehouse Management System API Documentation",
		ServerHeader: "Fiber",
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(helmet.New())

	// Configure CORS based on origins
	corsConfig := cors.Config{
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token", "X-Requested-With"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		ExposeHeaders: []string{"Content-Length", "Content-Type"},
		MaxAge:        86400, // 24 hours
	}

	// If origins contain wildcard, don't use credentials
	if len(cfg.CorsOrigins) == 1 && cfg.CorsOrigins[0] == "*" {
		corsConfig.AllowOrigins = []string{"*"}
		corsConfig.AllowCredentials = false
	} else {
		// Use custom origin validator to support wildcard patterns
		corsConfig.AllowOriginsFunc = func(origin string) bool {
			// Check each configured origin
			for _, allowedOrigin := range cfg.CorsOrigins {
				// Exact match
				if origin == allowedOrigin {
					return true
				}
				// Pattern match (e.g., http://192.168.41.*:8081)
				if matchOriginPattern(allowedOrigin, origin) {
					return true
				}
			}
			return false
		}
		corsConfig.AllowCredentials = true
	}

	app.Use(cors.New(corsConfig))
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 60 * time.Second,
	}))

	// Setup routes
	routes.SetupRoutes(app, cfg, database.DB)

	// Start server
	log.Println("════════════════════════════════════════════════════════════")
	log.Printf("✓ Server ready on port %s", cfg.Port)
	log.Printf("📊 Health check: %s/api/health", cfg.AppUrl)
	log.Printf("📚 API documentation: %s/rapidoc", cfg.AppUrl)
	log.Println("════════════════════════════════════════════════════════════")

	// port := fmt.Sprintf(":%s", cfg.Port)
	// log.Printf("Server starting on port %s", port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
