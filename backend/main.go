package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"sinapsis-backend/config"
	"sinapsis-backend/db"
	"sinapsis-backend/routes"
)

func main() {
	loadLocalEnv()

	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// timezone: todas las sesiones usan hora de Colombia para que "hoy"
	// (CURRENT_DATE / NOW()) coincida con la agenda y las citas del día.
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&timezone=America/Bogota",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("database connected successfully")

	// Carpeta para anexos (HU-07); persiste en el volumen 'uploads'.
	if err := os.MkdirAll(cfg.UploadsDir, 0o755); err != nil {
		log.Fatalf("no se pudo crear el directorio de uploads: %v", err)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	routes.Setup(r, pool, cfg)

	log.Printf("server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func loadLocalEnv() {
	for _, path := range []string{".env", "../.env", "backend/.env"} {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err != nil {
				log.Printf("warning: could not load %s: %v", path, err)
			}
			return
		}
	}
}
