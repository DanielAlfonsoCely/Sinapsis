package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"sinapsis-backend/config"
	"sinapsis-backend/db"
	"sinapsis-backend/routes"
)

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
    	log.Fatalf("database connection failed: %v", err)
	} else {
		defer pool.Close()
		log.Println("database connected successfully")
	}
	_ = pool

	r := gin.Default()
	routes.Setup(r)

	log.Printf("server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
