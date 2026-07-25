package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"sinapsis-backend/config"
	"sinapsis-backend/db"
	"sinapsis-backend/queue"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

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

	// Conexión AMQP con el microservicio de IA (opcional: si RABBITMQ_URL no está
	// configurada, el servidor arranca en modo degradado sin IA).
	var amqpConn *queue.Connection
	var publisher *queue.Publisher
	if cfg.RabbitMQURL != "" {
		amqpConn, err = connectAMQPWithRetry(cfg, 5, 2*time.Second)
		if err != nil {
			log.Printf("amqp: no se pudo conectar tras reintentos, IA deshabilitada: %v", err)
		} else {
			defer amqpConn.Close()
			publisher = queue.NewPublisher(amqpConn.Channel, cfg.RabbitMQRequestQueue)
			consumer := queue.NewConsumer(amqpConn.Channel, cfg.RabbitMQResultQueue, pool)
			go consumer.Start(ctx)
		}
	} else {
		log.Println("RABBITMQ_URL no configurada, módulo de IA deshabilitado")
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Disposition"},
		AllowCredentials: true,
	}))

	routes.Setup(r, pool, cfg, publisher)

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

func connectAMQPWithRetry(cfg *config.Config, maxAttempts int, delay time.Duration) (*queue.Connection, error) {
	var lastErr error
	for i := range maxAttempts {
		if i > 0 {
			time.Sleep(delay)
		}
		conn, err := queue.Connect(
			cfg.RabbitMQURL,
			cfg.RabbitMQResultExchange,
			cfg.RabbitMQResultRoutingKey,
			cfg.RabbitMQResultQueue,
		)
		if err == nil {
			log.Printf("amqp: conectado en intento %d/%d", i+1, maxAttempts)
			return conn, nil
		}
		lastErr = err
		log.Printf("amqp: intento %d/%d falló: %v", i+1, maxAttempts, err)
	}
	return nil, lastErr
}
