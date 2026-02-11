package routes

import (
	"fmt"
	"livo-fiber-backend/config"
	"livo-fiber-backend/controllers"
	"livo-fiber-backend/middleware"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, cfg *config.Config, db *gorm.DB) {

	// Controllers
	authController := controllers.NewAuthController(cfg, db)
	userController := controllers.NewUserController(db)
	roleController := controllers.NewRoleController(db)
	boxController := controllers.NewBoxController(db)
	channelController := controllers.NewChannelController(db)
	expeditionController := controllers.NewExpeditionController(db)
	storeController := controllers.NewStoreController(db)
	productController := controllers.NewProductController(db)
	orderController := controllers.NewOrderController(db)
	qcRibbonController := controllers.NewQCRibbonController(db)
	qcOnlineController := controllers.NewQCOnlineController(db)
	outboundController := controllers.NewOutboundController(db)
	ribbonFlowController := controllers.NewRibbonFlowController(db)
	onlineFlowController := controllers.NewOnlineFlowController(db)
	reportController := controllers.NewReportController(db)
	lostFoundController := controllers.NewLostFoundController(db)
	returnController := controllers.NewReturnController(db)
	returnPickedOrderController := controllers.NewPickedOrderController(db)
	complainController := controllers.NewComplainController(db)
	mobileChannelController := controllers.NewMobileChannelController(db)
	mobileStoreController := controllers.NewMobileStoreController(db)
	mobileReturnController := controllers.NewMobileReturnController(db)
	mobileOrderController := controllers.NewMobileOrderController(db)
	attendanceController := controllers.NewAttendanceController(db)
	mobileAttendanceController := controllers.NewMobileAttendanceController(db)
	locationController := controllers.NewLocationController(db)

	// Public routes
	api := app.Group("/api")

	// Health check
	api.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"Aplication": "Livotech Warehouse Management System API Documentation",
			"Version":    "1.0.0",
			"message":    "Health check successful",
			"status":     "ok",
			"Time":       time.Now().Format("02-01-2006 15:04:05"),
		})
	})

	// API Documentation routes - Serve static swagger files
	app.Get("/docs/swagger.json", func(c fiber.Ctx) error {
		return c.SendFile("./docs/swagger.json")
	})

	app.Get("/docs/swagger.yaml", func(c fiber.Ctx) error {
		return c.SendFile("./docs/swagger.yaml")
	})

	// Swagger UI HTML page
	app.Get("/docs", func(c fiber.Ctx) error {
		// Dynamic URLs based on the request
		scheme := "http"
		if c.Protocol() == "https" {
			scheme = "https"
		}
		host := c.Request().Host()
		url := fmt.Sprintf("%s://%s", scheme, host)

		html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="description" content="SwaggerUI" />
  <title>Livo API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js" crossorigin></script>
<script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
      url: "` + url + `/docs/swagger.yaml",
      dom_id: '#swagger-ui',
    });
  };
</script>
</body>
</html>`
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	})

	// RapiDoc HTML page
	app.Get("/rapidoc", func(c fiber.Ctx) error {
		// Dynamic URLs based on the request
		scheme := "http"
		if c.Protocol() == "https" {
			scheme = "https"
		}
		host := c.Request().Host()
		baseURL := fmt.Sprintf("%s://%s", scheme, host)
		specURL := fmt.Sprintf("%s/docs/swagger.yaml", baseURL)

		html := `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Livo API Documentation</title>
  <script type="module" src="https://unpkg.com/rapidoc/dist/rapidoc-min.js"></script>
</head>
<body>
  <rapi-doc
        spec-url="` + specURL + `"
        theme="dark"
        bg-color="#1a1a1a"
        text-color="#f0f0f0"
        primary-color="#4caf50"
        nav-bg-color="#2d2d2d"
        nav-text-color="#ffffff"
        nav-hover-bg-color="#404040"
				sort-endpoints-by="path"
        render-style="focused"
        layout="column"
        schema-style="tree"
        show-header="true"
				show-components="true"
        show-info="true"
        allow-try="true"
        allow-authentication="true"
        allow-spec-url-load="false"
        allow-spec-file-load="false"
        allow-search="true"
        allow-advanced-search="true"
        show-method-in-nav-bar="as-colored-block"
        use-path-in-nav-bar="true"
        response-area-height="400px"
				default-api-server="` + baseURL + `"
				show-curl-before-try="true"
        default-schema-tab="model"
        schema-expand-level="2"
        schema-description-expanded="true"
        schema-hide-read-only="never"
        schema-hide-write-only="never"
        fetch-credentials="include"
        heading-text="Livotech Warehouse Management System API Documentation"
        goto-path=""
        fill-request-fields-with-example="true"
        persist-auth="true"
    >
        <img 
            slot="logo" 
            src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%234caf50'%3E%3Cpath d='M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z'/%3E%3C/svg%3E"
            style="width: 40px; height: 40px; margin-right: 10px;"
        />
    </rapi-doc>
</body>
</html>`
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	})

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", authController.Register)
	auth.Post("/login", authController.Login)
	auth.Post("/refresh", authController.RefreshToken)

	// Attendances routes (public)
	attendances := api.Group("/attendances")
	attendances.Post("/search/face", attendanceController.SearchUsersByFace)
	attendances.Post("/checkin/face", attendanceController.CheckInUserByFace)
	attendances.Put("/checkout/face", attendanceController.CheckOutUserByFace)
	attendances.Post("/checkin/manual", attendanceController.CheckInUserManual)
	attendances.Put("/checkout/manual", attendanceController.CheckOutUserManual)

	// Mobile Returns routes (public)
	mobileReturns := api.Group("/mobile-returns")
	mobileReturns.Get("/channels", mobileChannelController.GetMobileChannels)
	mobileReturns.Get("/stores", mobileStoreController.GetMobileStores)
	mobileReturns.Get("/", mobileReturnController.GetMobileReturns)
	mobileReturns.Get("/:id", mobileReturnController.GetMobileReturn)
	mobileReturns.Post("/", mobileReturnController.CreateMobileReturn)

	// CSRF token endpoint for web clients
	auth.Get("/csrf-token", middleware.CSRFMiddleware(), func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"csrf_token": c.Locals("csrf_token"),
		})
	})

	// Redirect root to rapidoc
	app.Get("/", func(c fiber.Ctx) error {
		return c.Redirect().Status(fiber.StatusMovedPermanently).To("/rapidoc")
	})

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg))

	// Note: CSRF middleware removed for API clients (HTTPie, Postman, mobile apps)
	// If you need CSRF protection for web clients, apply it selectively to specific routes
	// protected.Use(middleware.CSRFMiddleware())

	// Auth protected routes
	protectedAuth := protected.Group("/auth")
	protectedAuth.Post("/logout", authController.Logout)

	// Mobile Attendance routes
	mobileAttendance := protected.Group("/mobile-attendances")
	mobileAttendance.Post("/face-verify", mobileAttendanceController.VerifyUserFace)
	mobileAttendance.Post("/checkin/face", mobileAttendanceController.MobileCheckInUserByFace)
	mobileAttendance.Put("/checkout/face", mobileAttendanceController.MobileCheckOutUserByFace)

	// User routes
	users := protected.Group("/users")
	users.Get("/", userController.GetUsers)
	users.Get("/:id", userController.GetUser)
	users.Post("/", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), userController.CreateUser)
	users.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), userController.UpdateUser)
	users.Put("/:id/password", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), userController.UpdatePassword)
	users.Delete("/:id", middleware.RoleMiddleware([]string{"developer"}), userController.DeleteUser)
	users.Post("/:id/roles", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), userController.AssignRole)
	users.Delete("/:id/roles", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), userController.RemoveRole)
	users.Post("/:id/face-register", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), userController.RegisterUserFace)
	users.Get("/:id/sessions", userController.GetSessions)

	// Role routes
	roles := protected.Group("/roles")
	roles.Get("/", roleController.GetRoles)
	roles.Get("/:id", roleController.GetRole)
	roles.Post("/", middleware.RoleMiddleware([]string{"admin", "developer"}), roleController.CreateRole)
	roles.Put("/:id", middleware.RoleMiddleware([]string{"admin", "developer"}), roleController.UpdateRole)
	roles.Delete("/:id", middleware.RoleMiddleware([]string{"admin", "developer"}), roleController.DeleteRole)

	// Box routes
	boxRoutes := protected.Group("/boxes")
	boxRoutes.Get("/", boxController.GetBoxes)
	boxRoutes.Get("/:id", boxController.GetBox)
	boxRoutes.Post("/", middleware.RoleMiddleware([]string{"developer", "superadmin"}), boxController.CreateBox)
	boxRoutes.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin"}), boxController.UpdateBox)
	boxRoutes.Delete("/:id", middleware.RoleMiddleware([]string{"developer"}), boxController.DeleteBox)

	// Channel routes
	channelRoutes := protected.Group("/channels")
	channelRoutes.Get("/", channelController.GetChannels)
	channelRoutes.Get("/:id", channelController.GetChannel)
	channelRoutes.Post("/", middleware.RoleMiddleware([]string{"developer", "superadmin"}), channelController.CreateChannel)
	channelRoutes.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin"}), channelController.UpdateChannel)
	channelRoutes.Delete("/:id", middleware.RoleMiddleware([]string{"developer"}), channelController.DeleteChannel)

	// Expedition routes
	expeditionRoutes := protected.Group("/expeditions")
	expeditionRoutes.Get("/", expeditionController.GetExpeditions)
	expeditionRoutes.Get("/:id", expeditionController.GetExpedition)
	expeditionRoutes.Post("/", middleware.RoleMiddleware([]string{"developer", "superadmin"}), expeditionController.CreateExpedition)
	expeditionRoutes.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin"}), expeditionController.UpdateExpedition)
	expeditionRoutes.Delete("/:id", middleware.RoleMiddleware([]string{"developer"}), expeditionController.DeleteExpedition)

	// Store routes
	storeRoutes := protected.Group("/stores")
	storeRoutes.Get("/", storeController.GetStores)
	storeRoutes.Get("/:id", storeController.GetStore)
	storeRoutes.Post("/", middleware.RoleMiddleware([]string{"developer", "superadmin"}), storeController.CreateStore)
	storeRoutes.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin"}), storeController.UpdateStore)
	storeRoutes.Delete("/:id", middleware.RoleMiddleware([]string{"developer"}), storeController.DeleteStore)

	// Product routes
	productRoutes := protected.Group("/products")
	productRoutes.Get("/", productController.GetProducts)
	productRoutes.Get("/:id", productController.GetProduct)
	productRoutes.Post("/", middleware.RoleMiddleware([]string{"developer", "superadmin", "admin", "warehouse"}), productController.CreateProduct)
	productRoutes.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin", "admin", "warehouse"}), productController.UpdateProduct)
	productRoutes.Delete("/:id", middleware.RoleMiddleware([]string{"developer"}), productController.DeleteProduct)

	// Order routes
	orderRoutes := protected.Group("/orders")
	orderRoutes.Get("/", orderController.GetOrders)
	orderRoutes.Get("/:id", orderController.GetOrder)
	orderRoutes.Put("/:id/status/qc-process", orderController.QCProcessStatusUpdate)
	orderRoutes.Put("/:id/status/picking-completed", orderController.PickingCompletedStatusUpdate)

	// Order router for admin
	orderRoutes.Post("/", middleware.RoleMiddleware([]string{"developer", "superadmin", "admin"}), orderController.CreateOrder)
	orderRoutes.Post("/bulk", middleware.RoleMiddleware([]string{"developer", "superadmin", "admin"}), orderController.BulkCreateOrders)
	orderRoutes.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin", "admin"}), orderController.UpdateOrder)
	orderRoutes.Put("/:id/duplicate", middleware.RoleMiddleware([]string{"developer", "superadmin", "admin"}), orderController.DuplicateOrder)
	orderRoutes.Put("/:id/cancel", middleware.RoleMiddleware([]string{"developer", "superadmin", "admin"}), orderController.CancelOrder)

	// Order router for coordinator
	orderRoutes.Post("/assign-picker", middleware.RoleMiddleware([]string{"developer", "superadmin", "coordinator"}), orderController.AssignPicker)
	orderRoutes.Put("/:id/pending-picking", middleware.RoleMiddleware([]string{"developer", "superadmin", "coordinator"}), orderController.PendingPickingOrders)
	orderRoutes.Get("/assigned", middleware.RoleMiddleware([]string{"developer", "superadmin", "coordinator"}), orderController.GetAssignedOrders)

	// Ribbon routes
	qcRibbonRoutes := protected.Group("/ribbons")
	// QC ribbon routes
	qcRibbonRoutes.Get("/qc-ribbons/chart", qcRibbonController.GetChartQCRibbons)
	qcRibbonRoutes.Get("/qc-ribbons", qcRibbonController.GetQCRibbons)
	qcRibbonRoutes.Get("/qc-ribbons/:id", qcRibbonController.GetQCRibbon)
	qcRibbonRoutes.Post("/qc-ribbons/start", qcRibbonController.QCRibbonStart)
	qcRibbonRoutes.Put("/qc-ribbons/:id/validate", qcRibbonController.ValidateQCRibbonProduct)
	qcRibbonRoutes.Put("/qc-ribbons/:id/complete", qcRibbonController.CompleteQcRibbon)
	qcRibbonRoutes.Put("/qc-ribbons/:id/pending", qcRibbonController.PendingQCRibbon)

	// Ribbon flow routes
	qcRibbonRoutes.Get("/flows", ribbonFlowController.GetRibbonFlows)
	qcRibbonRoutes.Get("/flows/:trackingNumber", ribbonFlowController.GetRibbonFlow)

	// QCOnline routes
	qcOnlineRoutes := protected.Group("/onlines")
	// QC online routes
	qcOnlineRoutes.Get("/qc-onlines/chart", qcOnlineController.GetChartQCOnlines)
	qcOnlineRoutes.Get("/qc-onlines/", qcOnlineController.GetQCOnlines)
	qcOnlineRoutes.Get("/qc-onlines/:id", qcOnlineController.GetQCOnline)
	qcOnlineRoutes.Post("/qc-onlines/start", qcOnlineController.QCOnlineStart)
	qcOnlineRoutes.Put("/qc-onlines/:id/validate", qcOnlineController.ValidateQCOnlineProduct)
	qcOnlineRoutes.Put("/qc-onlines/:id/complete", qcOnlineController.CompleteQcOnline)
	qcOnlineRoutes.Put("/qc-onlines/:id/pending", qcOnlineController.PendingQCOnline)

	// Online flow routes
	qcOnlineRoutes.Get("/flows", onlineFlowController.GetOnlineFlows)
	qcOnlineRoutes.Get("/flows/:trackingNumber", onlineFlowController.GetOnlineFlow)

	// Outbound routes
	outboundRoutes := protected.Group("/outbounds")
	outboundRoutes.Get("/", outboundController.GetOutbounds)
	// Chart Outbound routes
	outboundRoutes.Get("/chart", outboundController.GetChartOutbounds)
	outboundRoutes.Get("/:id", outboundController.GetOutbound)
	outboundRoutes.Post("/", outboundController.CreateOutbound)
	outboundRoutes.Put("/:id", outboundController.UpdateOutbound)

	// Report routes
	reportRoutes := protected.Group("/reports")
	reportRoutes.Get("/boxes", reportController.GetBoxReports)
	reportRoutes.Get("/outbounds", reportController.GetOutboundReports)
	reportRoutes.Get("/returns", reportController.GetReturnReports)
	reportRoutes.Get("/complains", reportController.GetComplainReports)
	reportRoutes.Get("/user-fees", reportController.GetUserFeeReports)

	// Lost and Found routes
	lostFoundRoutes := protected.Group("/lost-founds")
	lostFoundRoutes.Get("/", lostFoundController.GetLostfounds)
	lostFoundRoutes.Get("/:id", lostFoundController.GetLostfound)
	lostFoundRoutes.Post("/", lostFoundController.CreateLostfound)
	lostFoundRoutes.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin", "coordinator", "admin"}), lostFoundController.UpdateLostfound)
	lostFoundRoutes.Delete("/:id", middleware.RoleMiddleware([]string{"developer"}), lostFoundController.DeleteLostfound)

	// Return routes
	returnRoutes := protected.Group("/returns")
	returnRoutes.Get("/", returnController.GetReturns)
	returnRoutes.Get("/:id", returnController.GetReturn)
	returnRoutes.Post("/", returnController.CreateReturn)
	returnRoutes.Put("/:id", returnController.UpdateReturn)

	// Picked Order routes
	pickedOrderRoutes := protected.Group("/picked-orders")
	pickedOrderRoutes.Get("/", returnPickedOrderController.GetPickedOrders)
	pickedOrderRoutes.Get("/:id", returnPickedOrderController.GetPickedOrder)

	// Complain routes
	complainRoutes := protected.Group("/complains")
	complainRoutes.Get("/", complainController.GetComplains)
	complainRoutes.Get("/:id", complainController.GetComplain)
	complainRoutes.Post("/", complainController.CreateComplain)
	complainRoutes.Put("/:id", complainController.UpdateComplain)
	complainRoutes.Put("/:id/check", complainController.UpdateComplainCheck)

	// Mobile Orders routes
	mobileOrders := api.Group("/mobile-orders")
	mobileOrders.Get("/my-picking-orders", mobileOrderController.GetMyPickingOrders)
	mobileOrders.Get("/my-picking-orders/:id", mobileOrderController.GetMyPickingOrder)
	mobileOrders.Put("/my-picking-orders/:id/picked", mobileOrderController.UpdatePickedOrder)
	mobileOrders.Put("/my-picking-orders/:id/complete", mobileOrderController.CompletePickingOrder)
	mobileOrders.Put("/my-picking-orders/:id/pending", mobileOrderController.PendingPickOrder)
	mobileOrders.Put("/bulk-assign-picker", middleware.RoleMiddleware([]string{"developer", "superadmin", "coordinator"}), mobileOrderController.BulkAssignPicker)
	mobileOrders.Get("/", middleware.RoleMiddleware([]string{"developer", "superadmin", "coordinator"}), mobileOrderController.GetMobilePickedOrders)
	mobileOrders.Get("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin", "coordinator"}), mobileOrderController.GetMobilePickedOrder)

	// Location routes
	locationRoutes := protected.Group("/locations")
	locationRoutes.Get("/", locationController.GetLocations)
	locationRoutes.Get("/:id", locationController.GetLocation)
	locationRoutes.Post("/", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), locationController.CreateLocation)
	locationRoutes.Put("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), locationController.UpdateLocation)
	locationRoutes.Delete("/:id", middleware.RoleMiddleware([]string{"developer", "superadmin", "hrd"}), locationController.DeleteLocation)

	// Attendance management routes (protected - developer and hrd only)
	attendanceManagement := protected.Group("/attendances")
	attendanceManagement.Get("/", middleware.RoleMiddleware([]string{"developer", "hrd"}), attendanceController.GetAttendances)
	attendanceManagement.Get("/:id", middleware.RoleMiddleware([]string{"developer", "hrd"}), attendanceController.GetAttendanceByID)

}
