package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"project_smt6/api"
	"project_smt6/db"
	"project_smt6/monitor"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env jika ada
	_ = godotenv.Load()

	log.Println("🌐 NetMonitor - Network Monitoring + Kepuasan Jaringan")
	log.Println("========================================================")

	// Koneksi database
	db.Connect()

	// Mulai ping monitor di background
	monitorSvc := monitor.NewMonitorService()
	monitorSvc.Start()

	// Setup HTTP router
	router := api.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 Server berjalan di http://localhost:%s\n", port)
		if err := router.Run(":" + port); err != nil {
			log.Fatalf("❌ Server gagal: %v", err)
		}
	}()

	<-quit
	log.Println("⏳ Menghentikan server...")
	monitorSvc.Stop()
	log.Println("✅ Server berhenti dengan aman")
}