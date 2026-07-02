package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/config"
	"sinapsis-backend/handlers"
)

type Handler struct {
	auth *handlers.AuthHandler
}

func Setup(r *gin.Engine, pool *pgxpool.Pool, cfg *config.Config) {
	h := &Handler{
		auth: handlers.NewAuthHandler(pool, cfg),
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.auth.Register)
			auth.POST("/login", h.auth.Login)
		}
	}
}
