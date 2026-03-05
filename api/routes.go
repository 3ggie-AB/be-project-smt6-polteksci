package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// CORS untuk React dev server
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	v1 := r.Group("/api")
	{
		// Targets (daftar IP yang di-ping)
		v1.GET("/targets", GetTargets)
		v1.POST("/targets", CreateTarget)
		v1.DELETE("/targets/:id", DeleteTarget)

		// Ping results
		v1.GET("/pings/latest", GetLatestPings)
		v1.GET("/pings/history", GetPingHistory)
		v1.GET("/pings/summary", GetPingSummary)

		// Angket kepuasan
		v1.POST("/surveys", SubmitSurvey)
		v1.GET("/surveys", GetSurveys)

		// Korelasi
		v1.GET("/correlation", GetCorrelation)
	}

	return r
}