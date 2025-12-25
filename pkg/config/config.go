package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	GigaChat GigaChatConfig
	OCR      OCRConfig
	RAG      RAGConfig
	Logger   LoggerConfig
}

type LoggerConfig struct {
	Level string
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	SecretKey  string
	Expiration time.Duration
	RefreshExp time.Duration
}

type GigaChatConfig struct {
	APIKey             string
	Scope              string
	InsecureSkipVerify bool
}

type OCRConfig struct {
	Provider string // Deprecated: now using GigaChat Vision API
	APIKey   string // Deprecated: now using GigaChat Vision API
}

type RAGConfig struct {
	EmbeddingModel string
	TopK           int
}

func Load() (*Config, error) {
	// Try to load .env file from current directory or project root
	envFiles := []string{".env", "../.env", "../../.env"}
	var loaded bool
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			loaded = true
			break
		}
	}

	// If no .env file found, continue with environment variables
	// This allows using environment variables directly (useful for Docker/K8s)
	if !loaded {
		// .env file is optional, continue with environment variables
	}

	readTimeout, _ := strconv.Atoi(getEnv("SERVER_READ_TIMEOUT", "30"))
	writeTimeout, _ := strconv.Atoi(getEnv("SERVER_WRITE_TIMEOUT", "30"))
	jwtExp, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	refreshExp, _ := strconv.Atoi(getEnv("JWT_REFRESH_EXPIRATION_HOURS", "168"))
	ragTopK, _ := strconv.Atoi(getEnv("RAG_TOP_K", "5"))
	insecureSkipVerify := getEnv("GIGACHAT_INSECURE_SKIP_VERIFY", "true") == "true"

	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5433"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "rag_iishka"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			SecretKey:  getEnv("JWT_SECRET_KEY", "your-secret-key-change-in-production"),
			Expiration: time.Duration(jwtExp) * time.Hour,
			RefreshExp: time.Duration(refreshExp) * time.Hour,
		},
		GigaChat: GigaChatConfig{
			APIKey:             getEnv("GIGACHAT_API_KEY", ""),
			Scope:              getEnv("GIGACHAT_SCOPE", "GIGACHAT_API_PERS"),
			InsecureSkipVerify: insecureSkipVerify,
		},
		OCR: OCRConfig{
			Provider: getEnv("OCR_PROVIDER", "tesseract"),
			APIKey:   getEnv("OCR_API_KEY", ""),
		},
		RAG: RAGConfig{
			EmbeddingModel: getEnv("RAG_EMBEDDING_MODEL", "text-embedding-ada-002"),
			TopK:           ragTopK,
		},
		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
