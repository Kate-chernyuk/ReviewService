package main

import (
	"PR/config"
	"PR/handlers"
	"PR/repository"
	"PR/service"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	db, err := config.InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	repo := repository.NewRepository(db)
	reviewService := service.NewReviewService(repo)
	handler := handlers.NewHandler(reviewService)

	r := gin.Default()

	r.POST("/team/add", handler.CreateTeam)
	r.GET("/team/get", handler.GetTeam)

	r.POST("/users/setIsActive", handler.SetUserActive)

	r.POST("/pullRequest/create", handler.CreatePR)
	r.POST("/pullRequest/merge", handler.MergePR)
	r.POST("/pullRequest/reassign", handler.ReassignReviewer)

	r.GET("/users/getReview", handler.GetUserReviews)

	r.GET("/stats/user", handler.GetUserStats)
	r.POST("/users/bulkDeactivate", handler.BulkDeactivateUsers)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
