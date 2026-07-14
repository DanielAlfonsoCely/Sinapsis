package config

import "os"

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	ServerPort string
	UploadsDir string

	// RabbitMQ — integración con el microservicio de IA
	RabbitMQURL             string
	RabbitMQRequestQueue    string
	RabbitMQResultExchange  string
	RabbitMQResultRoutingKey string
	RabbitMQResultQueue     string
}

func Load() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", ""),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", ""),
		JWTSecret:  getEnv("JWT_SECRET", ""),
		ServerPort: getEnv("SERVER_PORT", "8080"),
		UploadsDir: getEnv("UPLOADS_DIR", "/app/uploads"),

		RabbitMQURL:              getEnv("RABBITMQ_URL", ""),
		RabbitMQRequestQueue:     getEnv("RABBITMQ_REQUEST_QUEUE", "ai.analysis.requests"),
		RabbitMQResultExchange:   getEnv("RABBITMQ_RESULT_EXCHANGE", "sinapsis.ai"),
		RabbitMQResultRoutingKey: getEnv("RABBITMQ_RESULT_ROUTING_KEY", "ai.analysis.result"),
		RabbitMQResultQueue:      getEnv("RABBITMQ_RESULT_QUEUE", "backend.ai.results"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
