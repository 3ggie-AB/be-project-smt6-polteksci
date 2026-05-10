package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	authpkg "project_smt6/auth"
	"project_smt6/collector"
	"project_smt6/influx"
	"project_smt6/internal/config"
	"project_smt6/ml"
	mysqlstore "project_smt6/mysql"
	"project_smt6/service"
	"project_smt6/websocket"
)

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	logger.Info("Startup checklist")
	store, err := mysqlstore.New(ctx, cfg.MySQL, logger)
	if err != nil {
		logger.Error("[FAIL] MySQL connection failed", "database", cfg.MySQL.Database, "detail", err)
		return err
	}
	if err := store.Ping(ctx); err != nil {
		logger.Error("[FAIL] MySQL ping failed", "database", cfg.MySQL.Database, "detail", err)
		return err
	}
	logger.Info("[OK] MySQL connected", "host", cfg.MySQL.Host, "port", cfg.MySQL.Port, "database", cfg.MySQL.Database)
	logger.Info("[OK] MySQL schema migrated", "database", cfg.MySQL.Database)
	defer func() {
		if err := store.Close(); err != nil {
			logger.Error("MySQL close failed", "detail", err)
		}
	}()

	deviceRepo := mysqlstore.DeviceRepositoryAdapter{Store: store}
	notificationRepo := mysqlstore.NotificationRepositoryAdapter{Store: store}
	metricWriter := influx.NewWriter(cfg.Influx, logger)
	checkCtx, cancelCheck := context.WithTimeout(ctx, 3*time.Second)
	if metricWriter.Enabled() {
		if err := metricWriter.Ping(checkCtx); err != nil {
			logger.Warn("[FAIL] InfluxDB health check failed", "url", cfg.Influx.URL, "org", cfg.Influx.Org, "bucket", cfg.Influx.Bucket, "detail", err)
		} else {
			logger.Info("[OK] InfluxDB connected", "url", cfg.Influx.URL, "org", cfg.Influx.Org, "bucket", cfg.Influx.Bucket)
		}
	} else {
		logger.Warn("[WARN] InfluxDB disabled", "reason", "INFLUX_URL, INFLUX_TOKEN, INFLUX_ORG, or INFLUX_BUCKET is empty")
	}
	cancelCheck()

	writerCtx, cancelWriter := context.WithCancel(context.Background())
	metricWriter.Start(writerCtx)
	defer func() {
		cancelWriter()
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := metricWriter.Close(closeCtx); err != nil {
			logger.Error("InfluxDB writer close failed", "detail", err)
		}
	}()

	broker := websocket.NewBroker(logger)
	featureEngine := ml.NewFeatureEngine(120)
	authSvc := authpkg.NewService(cfg.Auth, store)
	deviceSvc := service.NewDeviceService(deviceRepo)

	runCtx, cancelCollectors := context.WithCancel(ctx)
	defer cancelCollectors()
	var collectorWG sync.WaitGroup
	startCollector := func(name string, fn func(context.Context)) {
		collectorWG.Add(1)
		go func() {
			defer collectorWG.Done()
			logger.Info("Starting collector", "name", name)
			fn(runCtx)
		}()
	}

	active := collector.NewActiveEngine(deviceRepo, metricWriter, broker, featureEngine, cfg.Monitoring, logger)
	ruijie := collector.NewRuijieCollector(cfg.Ruijie, metricWriter, broker, featureEngine, logger)
	syslog := collector.NewSyslogReceiver(cfg.Syslog, metricWriter, broker, logger)
	snmp := collector.NewSNMPCollector(cfg.SNMP, deviceRepo, metricWriter, featureEngine, logger)
	logger.Info("[OK] active monitoring configured", "ping_interval", cfg.Monitoring.PingInterval.String(), "tcp_interval", cfg.Monitoring.TCPInterval.String(), "ping_workers", cfg.Monitoring.PingWorkers, "tcp_workers", cfg.Monitoring.TCPWorkers)
	if cfg.Ruijie.BaseURL == "" {
		logger.Warn("[WARN] Ruijie collector disabled", "reason", "RUIJIE_BASE_URL is empty")
	} else {
		logger.Info("[OK] Ruijie collector configured", "base_url", cfg.Ruijie.BaseURL, "interval", cfg.Ruijie.PollInterval.String())
	}
	if cfg.Syslog.Enabled {
		logger.Info("[OK] Syslog receiver configured", "address", cfg.Syslog.Address)
	} else {
		logger.Warn("[WARN] Syslog receiver disabled")
	}
	if cfg.SNMP.Enabled {
		logger.Info("[OK] SNMP collector configured", "interval", cfg.SNMP.PollInterval.String(), "port", cfg.SNMP.Port)
	} else {
		logger.Warn("[WARN] SNMP collector disabled")
	}
	startCollector("active", active.Run)
	if cfg.Ruijie.BaseURL != "" {
		startCollector("ruijie", ruijie.Run)
	}
	if cfg.Syslog.Enabled {
		startCollector("syslog", syslog.Run)
	}
	if cfg.SNMP.Enabled {
		startCollector("snmp", snmp.Run)
	}

	router := NewRouter(RouterDeps{
		Config:        cfg,
		Logger:        logger,
		Auth:          authSvc,
		Devices:       deviceSvc,
		Users:         store,
		Notifications: notificationRepo,
		Realtime:      broker,
		Features:      featureEngine,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("[OK] HTTP server started", "url", "http://localhost:"+cfg.Server.Port, "addr", server.Addr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil {
			cancelCollectors()
			collectorWG.Wait()
			return fmt.Errorf("http server failed: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cancelCollectors()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	collectorWG.Wait()
	logger.Info("[OK] Application stopped gracefully")
	return nil
}
