package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"project_smt6/app/services/monitoring"
	"project_smt6/config"
	"project_smt6/database"
	"project_smt6/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func Run(ctx context.Context, cfg config.AppConfig, logger *slog.Logger) error {
	logger.Info("Startup checklist")

	db, err := database.Connect(ctx, cfg.MySQL)
	if err != nil {
		logger.Error("❌ MySQL connection failed", "detail", err.Error())
		return err
	}
	logger.Info("✅ MySQL connected", "host", cfg.MySQL.Host, "port", cfg.MySQL.Port, "database", cfg.MySQL.Database)
	logger.Info("✅ GORM AutoMigrate completed")

	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	app := fiber.New(fiber.Config{
		AppName:     cfg.Name,
		ReadTimeout: 10 * time.Second,
	})
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     joinOrigins(cfg.CORS),
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	routes.Register(app, db, cfg)
	monitoring.NewService(db, cfg.Monitoring, logger).Start(ctx)

	serverErr := make(chan error, 1)
	go func() {
		addr := ":" + cfg.Port
		logger.Info("✅ HTTP server started", "url", "http://localhost:"+cfg.Port, "addr", addr)
		serverErr <- app.Listen(addr)
	}()

	select {
	case <-ctx.Done():
		logger.Info("Shutting down application")
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server failed: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown fiber: %w", err)
	}
	logger.Info("✅ Application stopped gracefully")
	return nil
}

func joinOrigins(origins []string) string {
	if len(origins) == 0 {
		return "*"
	}
	out := origins[0]
	for _, origin := range origins[1:] {
		out += "," + origin
	}
	return out
}
