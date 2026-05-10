package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	authpkg "project_smt6/auth"
	"project_smt6/domain"
	"project_smt6/internal/config"
	"project_smt6/middleware"
	"project_smt6/ml"
	"project_smt6/repository"
	"project_smt6/service"
	"project_smt6/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type RouterDeps struct {
	Config        config.Config
	Logger        *slog.Logger
	Auth          *authpkg.Service
	Devices       *service.DeviceService
	Users         repository.UserRepository
	Notifications repository.NotificationRepository
	Realtime      *websocket.Broker
	Features      *ml.FeatureEngine
}

func NewRouter(deps RouterDeps) *gin.Engine {
	if deps.Config.Server.Environment == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     deps.Config.Server.AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":        "NetMonitor API",
			"description": "Backend API untuk observability dan monitoring jaringan: ping, TCP health check, syslog, SNMP, Ruijie telemetry, realtime SSE, dan ML-ready metrics.",
			"status":      "running",
			"endpoints": gin.H{
				"health":  "/healthz",
				"login":   "/api/auth/login",
				"stream":  "/api/stream",
				"devices": "/api/devices",
			},
		})
	})

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
			"checks": gin.H{
				"mysql": gin.H{
					"status":   "ok",
					"database": deps.Config.MySQL.Database,
				},
				"influxdb": gin.H{
					"status": enabledStatus(deps.Config.Influx.Token != ""),
					"url":    deps.Config.Influx.URL,
					"bucket": deps.Config.Influx.Bucket,
				},
				"collectors": gin.H{
					"active": "enabled",
					"ruijie": enabledStatus(deps.Config.Ruijie.BaseURL != ""),
					"syslog": enabledStatus(deps.Config.Syslog.Enabled),
					"snmp":   enabledStatus(deps.Config.SNMP.Enabled),
				},
			},
		})
	})

	api := r.Group("/api")
	api.POST("/auth/login", login(deps))
	api.POST("/login", login(deps))

	protected := api.Group("")
	protected.Use(middleware.RequireAuth(deps.Auth))
	protected.GET("/me", currentUser())
	protected.GET("/stream", deps.Realtime.ServeSSE())
	protected.GET("/devices", listDevices(deps))
	protected.POST("/devices", middleware.RequireRoles(domain.RoleSuperAdmin, domain.RoleAdmin), createDevice(deps))
	protected.DELETE("/devices/:id", middleware.RequireRoles(domain.RoleSuperAdmin, domain.RoleAdmin), deleteDevice(deps))
	protected.GET("/notifications", listNotifications(deps))
	protected.POST("/notifications/:id/read", markNotificationRead(deps))
	protected.GET("/ml/features/:device_id", getFeatureVector(deps))

	return r
}

func login(deps RouterDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req authpkg.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email/password tidak valid"})
			return
		}
		user, token, err := deps.Auth.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "email atau password salah"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "login berhasil",
			"token":   token,
			"user":    user,
		})
	}
}

func currentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":      middleware.UserID(c),
			"email":        c.GetString(middleware.ContextEmail),
			"role":         c.MustGet(middleware.ContextRole),
			"workspace_id": middleware.WorkspaceID(c),
		})
	}
}

func listDevices(deps RouterDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		devices, err := deps.Devices.List(c.Request.Context(), middleware.WorkspaceID(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list devices"})
			return
		}
		c.JSON(http.StatusOK, devices)
	}
}

func createDevice(deps RouterDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var device domain.Device
		if err := c.ShouldBindJSON(&device); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := deps.Devices.Create(c.Request.Context(), middleware.WorkspaceID(c), &device); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, device)
	}
}

func deleteDevice(deps RouterDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := deps.Devices.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete device"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "device deleted"})
	}
}

func listNotifications(deps RouterDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceID := middleware.WorkspaceID(c)
		if workspaceID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "workspace is required"})
			return
		}
		notifications, err := deps.Notifications.ListUnread(c.Request.Context(), *workspaceID, middleware.UserID(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notifications"})
			return
		}
		c.JSON(http.StatusOK, notifications)
	}
}

func markNotificationRead(deps RouterDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := deps.Notifications.MarkRead(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification as read"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
	}
}

func getFeatureVector(deps RouterDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "device_id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		vector, ok := deps.Features.VectorFor(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "feature vector not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"features":   vector,
			"onnx_input": vector.ONNXInput(),
		})
	}
}

func parseUintParam(c *gin.Context, name string) (uint, error) {
	raw := c.Param(name)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return uint(id), nil
}

func enabledStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
