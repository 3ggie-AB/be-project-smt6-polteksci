package routes

import (
	"network-monitor/internal/config"
	"network-monitor/internal/database"
	"network-monitor/internal/handlers"
	"network-monitor/internal/middleware"
	"network-monitor/internal/services"

	"github.com/gofiber/fiber/v2"
)

type RouteInfo struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	Route  string `json:"route"`
}

func Setup(app *fiber.App, cfg *config.Config) {
	db := database.GetDB()
	ch := database.GetCH()

	pingService := services.NewPingService(ch, cfg.ClickHouseDatabase)
	snmpService := services.NewSNMPService(ch, cfg.ClickHouseDatabase)

	authHandler := handlers.NewAuthHandler(db, cfg)
	userHandler := handlers.NewUserHandler(db)
	monitoringHandler := handlers.NewMonitoringHandler(db, pingService, snmpService)
	feedbackHandler := handlers.NewFeedbackHandler(db)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success":  true,
			"message":  "Daftar route Network Monitor API",
			"base_url": "/api/v1",
			"docs":     "API_DOCS.md",
			"data":     apiRoutes(),
		})
	})

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Network Monitor API berjalan dengan normal",
			"version": "1.0.0",
		})
	})

	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/register", authHandler.Register)

	protected := api.Group("", middleware.JWTMiddleware(cfg.JWTSecret))

	meGroup := protected.Group("/auth")
	meGroup.Get("/me", authHandler.Me)
	meGroup.Put("/change-password", authHandler.ChangePassword)

	users := protected.Group("/users")
	users.Get("/", middleware.RequirePermission(db, "users:read"), userHandler.ListUsers)
	users.Post("/", middleware.RequirePermission(db, "users:write"), userHandler.CreateUser)
	users.Get("/:id", middleware.RequirePermission(db, "users:read"), userHandler.GetUser)
	users.Put("/:id", middleware.RequirePermission(db, "users:write"), userHandler.UpdateUser)
	users.Delete("/:id", middleware.RequirePermission(db, "users:delete"), userHandler.DeleteUser)
	users.Post("/:id/reset-password", middleware.RequireRole("atasan"), userHandler.ResetPassword)

	roles := protected.Group("/roles")
	roles.Get("/", middleware.RequirePermission(db, "roles:read"), userHandler.ListRoles)
	roles.Get("/:id", middleware.RequirePermission(db, "roles:read"), userHandler.GetRole)
	roles.Put("/:id/permissions", middleware.RequirePermission(db, "roles:write"), userHandler.UpdateRolePermissions)

	permissions := protected.Group("/permissions")
	permissions.Get("/", middleware.RequirePermission(db, "roles:read"), userHandler.ListPermissions)

	devices := protected.Group("/devices")
	devices.Get("/", middleware.RequirePermission(db, "devices:read"), monitoringHandler.ListDevices)
	devices.Post("/", middleware.RequirePermission(db, "devices:write"), monitoringHandler.CreateDevice)
	devices.Get("/:id", middleware.RequirePermission(db, "devices:read"), monitoringHandler.GetDevice)
	devices.Put("/:id", middleware.RequirePermission(db, "devices:write"), monitoringHandler.UpdateDevice)
	devices.Delete("/:id", middleware.RequirePermission(db, "devices:delete"), monitoringHandler.DeleteDevice)

	monitoring := protected.Group("/monitoring")
	monitoring.Post("/ping", middleware.RequirePermission(db, "monitoring:write"), monitoringHandler.PingCustom)
	monitoring.Post("/devices/:id/ping", middleware.RequirePermission(db, "monitoring:write"), monitoringHandler.PingDevice)
	monitoring.Get("/devices/:id/ping/history", middleware.RequirePermission(db, "monitoring:read"), monitoringHandler.GetPingHistory)
	monitoring.Post("/devices/:id/snmp", middleware.RequirePermission(db, "monitoring:write"), monitoringHandler.SNMPDevice)
	monitoring.Get("/devices/:id/snmp/history", middleware.RequirePermission(db, "monitoring:read"), monitoringHandler.GetSNMPHistory)
	monitoring.Get("/oids", middleware.RequirePermission(db, "monitoring:read"), monitoringHandler.GetOIDs)

	feedback := protected.Group("/feedback")
	feedback.Get("/", feedbackHandler.ListFeedbacks)
	feedback.Get("/stats", feedbackHandler.GetStats)
	feedback.Post("/", feedbackHandler.CreateFeedback)
	feedback.Get("/:id", feedbackHandler.GetFeedback)
	feedback.Put("/:id", feedbackHandler.UpdateFeedback)
	feedback.Delete("/:id", middleware.RequirePermission(db, "feedback:delete"), feedbackHandler.DeleteFeedback)
	feedback.Post("/:id/respond", middleware.RequirePermission(db, "feedback:respond"), feedbackHandler.RespondFeedback)
}

func apiRoutes() []RouteInfo {
	return []RouteInfo{
		{Name: "Health Check", Method: "GET", Route: "/api/v1/health"},
		{Name: "Login", Method: "POST", Route: "/api/v1/auth/login"},
		{Name: "Register", Method: "POST", Route: "/api/v1/auth/register"},
		{Name: "Profil Saya", Method: "GET", Route: "/api/v1/auth/me"},
		{Name: "Ganti Password", Method: "PUT", Route: "/api/v1/auth/change-password"},
		{Name: "Daftar User", Method: "GET", Route: "/api/v1/users"},
		{Name: "Buat User", Method: "POST", Route: "/api/v1/users"},
		{Name: "Detail User", Method: "GET", Route: "/api/v1/users/:id"},
		{Name: "Update User", Method: "PUT", Route: "/api/v1/users/:id"},
		{Name: "Hapus User", Method: "DELETE", Route: "/api/v1/users/:id"},
		{Name: "Reset Password User", Method: "POST", Route: "/api/v1/users/:id/reset-password"},
		{Name: "Daftar Role", Method: "GET", Route: "/api/v1/roles"},
		{Name: "Detail Role", Method: "GET", Route: "/api/v1/roles/:id"},
		{Name: "Update Permission Role", Method: "PUT", Route: "/api/v1/roles/:id/permissions"},
		{Name: "Daftar Permission", Method: "GET", Route: "/api/v1/permissions"},
		{Name: "Daftar Perangkat", Method: "GET", Route: "/api/v1/devices"},
		{Name: "Tambah Perangkat", Method: "POST", Route: "/api/v1/devices"},
		{Name: "Detail Perangkat", Method: "GET", Route: "/api/v1/devices/:id"},
		{Name: "Update Perangkat", Method: "PUT", Route: "/api/v1/devices/:id"},
		{Name: "Hapus Perangkat", Method: "DELETE", Route: "/api/v1/devices/:id"},
		{Name: "Ping Custom", Method: "POST", Route: "/api/v1/monitoring/ping"},
		{Name: "Ping Perangkat", Method: "POST", Route: "/api/v1/monitoring/devices/:id/ping"},
		{Name: "Histori Ping Perangkat", Method: "GET", Route: "/api/v1/monitoring/devices/:id/ping/history"},
		{Name: "SNMP Perangkat", Method: "POST", Route: "/api/v1/monitoring/devices/:id/snmp"},
		{Name: "Histori SNMP Perangkat", Method: "GET", Route: "/api/v1/monitoring/devices/:id/snmp/history"},
		{Name: "Daftar OID", Method: "GET", Route: "/api/v1/monitoring/oids"},
		{Name: "Daftar Feedback", Method: "GET", Route: "/api/v1/feedback"},
		{Name: "Statistik Feedback", Method: "GET", Route: "/api/v1/feedback/stats"},
		{Name: "Buat Feedback", Method: "POST", Route: "/api/v1/feedback"},
		{Name: "Detail Feedback", Method: "GET", Route: "/api/v1/feedback/:id"},
		{Name: "Update Feedback", Method: "PUT", Route: "/api/v1/feedback/:id"},
		{Name: "Hapus Feedback", Method: "DELETE", Route: "/api/v1/feedback/:id"},
		{Name: "Balas Feedback", Method: "POST", Route: "/api/v1/feedback/:id/respond"},
	}
}
