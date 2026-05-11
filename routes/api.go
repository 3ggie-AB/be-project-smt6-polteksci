package routes

import (
	"project_smt6/app/http/controllers"
	"project_smt6/app/http/middleware"
	"project_smt6/app/models"
	"project_smt6/config"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Register(app *fiber.App, db *gorm.DB, cfg config.AppConfig) {
	auth := controllers.NewAuthController(db, cfg)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name":        cfg.Name,
			"description": "Fiber API untuk monitoring jaringan, device inventory, alerting, notification, topology, dan ML observability.",
			"status":      "running",
			"endpoints": fiber.Map{
				"health":             "/healthz",
				"login":              "/api/auth/login",
				"users":              "/api/users",
				"sessions":           "/api/sessions",
				"devices":            "/api/devices",
				"device_status":      "/api/device-status",
				"monitoring_configs": "/api/monitoring-configs",
				"alerts":             "/api/alerts",
				"notifications":      "/api/notifications",
				"activity_logs":      "/api/activity-logs",
				"network_topology":   "/api/network-topology",
				"ml_predictions":     "/api/ml-predictions",
				"ml_anomalies":       "/api/ml-anomalies",
			},
		})
	})

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"stack":  "fiber + gorm + mysql",
			"mysql": fiber.Map{
				"host":     cfg.MySQL.Host,
				"port":     cfg.MySQL.Port,
				"database": cfg.MySQL.Database,
			},
		})
	})

	api := app.Group("/api")
	api.Post("/auth/login", auth.Login)
	api.Post("/login", auth.Login)

	protected := api.Group("", middleware.Auth(db))
	protected.Get("/auth/me", auth.Me)
	protected.Get("/me", auth.Me)
	protected.Post("/auth/logout", auth.Logout)

	registerUsers(protected, db)
	registerResource[models.Session](protected, db, "/sessions", "session")
	registerResource[models.Device](protected, db, "/devices", "device")
	registerResource[models.DeviceStatus](protected, db, "/device-status", "device status")
	registerResource[models.MonitoringConfig](protected, db, "/monitoring-configs", "monitoring config")
	registerResource[models.Alert](protected, db, "/alerts", "alert")
	registerResource[models.Notification](protected, db, "/notifications", "notification")
	registerResource[models.ActivityLog](protected, db, "/activity-logs", "activity log")
	registerResource[models.NetworkTopology](protected, db, "/network-topology", "network topology")
	registerResource[models.MLPrediction](protected, db, "/ml-predictions", "ml prediction")
	registerResource[models.MLAnomaly](protected, db, "/ml-anomalies", "ml anomaly")
}

func registerUsers(router fiber.Router, db *gorm.DB) {
	ctl := controllers.NewUserController(db)
	group := router.Group("/users")
	group.Get("/", ctl.Index)
	group.Post("/", middleware.Role(models.RoleSuperAdmin, models.RoleAdmin), ctl.Store)
	group.Get("/:id", ctl.Show)
	group.Put("/:id", middleware.Role(models.RoleSuperAdmin, models.RoleAdmin), ctl.Update)
	group.Patch("/:id", middleware.Role(models.RoleSuperAdmin, models.RoleAdmin), ctl.Update)
	group.Delete("/:id", middleware.Role(models.RoleSuperAdmin), ctl.Destroy)
}

func registerResource[T any](router fiber.Router, db *gorm.DB, path string, name string) {
	ctl := controllers.NewResourceController[T](db, name)
	group := router.Group(path)
	group.Get("/", ctl.Index)
	group.Post("/", middleware.Role(models.RoleSuperAdmin, models.RoleAdmin), ctl.Store)
	group.Get("/:id", ctl.Show)
	group.Put("/:id", middleware.Role(models.RoleSuperAdmin, models.RoleAdmin), ctl.Update)
	group.Patch("/:id", middleware.Role(models.RoleSuperAdmin, models.RoleAdmin), ctl.Update)
	group.Delete("/:id", middleware.Role(models.RoleSuperAdmin, models.RoleAdmin), ctl.Destroy)
}
