package main

import (
	"log"
	"os"

	"docker-gui-backend/internal/database"
	"docker-gui-backend/internal/docker"
	"docker-gui-backend/internal/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	db, err := database.NewDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatal("Failed to initialize Docker client:", err)
	}
	defer dockerClient.Close()

	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	containerHandler := handlers.NewContainerHandler(dockerClient, db)
	metricsHandler := handlers.NewMetricsHandler(dockerClient)
	imageHandler := handlers.NewImageHandler(dockerClient, db)

	api := r.Group("/api/v1")
	{
		containers := api.Group("/containers")
		{
			containers.GET("", containerHandler.ListContainers)
			containers.POST("/:id/start", containerHandler.StartContainer)
			containers.POST("/:id/stop", containerHandler.StopContainer)
			containers.POST("/:id/restart", containerHandler.RestartContainer)
			containers.DELETE("/:id", containerHandler.RemoveContainer)
			containers.GET("/:id/logs", containerHandler.GetContainerLogs)
			containers.GET("/:id/stats", containerHandler.GetContainerStats)
			containers.POST("/:id/action", containerHandler.PerformAction)
		}
		
		images := api.Group("/images")
		{
			images.GET("", imageHandler.ListImages)
			images.POST("/pull", imageHandler.PullImage)
			images.DELETE("/:id", imageHandler.RemoveImage)
			images.POST("/prune", imageHandler.PruneImages)
		}
		
		logs := api.Group("/logs")
		{
			logs.GET("", containerHandler.GetActivityLogs)
			logs.GET("/:id", containerHandler.GetContainerActivityLogs)
		}
		
		metrics := api.Group("/metrics")
		{
			metrics.GET("", metricsHandler.GetOverallMetrics)
			metrics.GET("/:id", metricsHandler.GetContainerMetrics)
			metrics.GET("/historical", metricsHandler.GetHistoricalMetrics)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}