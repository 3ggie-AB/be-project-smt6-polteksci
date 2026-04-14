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
		// Login endpoint (tidak perlu auth)
		v1.POST("/login", Login)
		v1.POST("/surveys", SubmitSurvey)

		// Endpoint yang perlu autentikasi
		protected := v1.Group("")
		protected.Use(AuthRequired())
		{
			// Targets (daftar IP yang di-ping)
			protected.GET("/targets", GetTargets)
			protected.POST("/targets", CreateTarget)
			protected.DELETE("/targets/:id", DeleteTarget)

			// Ping results
			protected.GET("/pings/latest", GetLatestPings)
			protected.GET("/pings/history", GetPingHistory)
			protected.GET("/pings/summary", GetPingSummary)

			// Angket kepuasan
			protected.GET("/surveys", GetSurveys)

			// Korelasi
			protected.GET("/correlation", GetCorrelation)
		}
	}

	return r
}