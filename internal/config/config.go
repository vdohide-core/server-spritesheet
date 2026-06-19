package config

import (
	"os"

	"github.com/joho/godotenv"
)

// AppConfig holds the application configuration loaded from environment variables.
var AppConfig Config

// Config represents the application configuration.
type Config struct {
	Port     string
	MongoURI string

	StorageId   string
	StoragePath string

	LogPath string // Path to rotating log file (env: LOG_PATH)
}

// Load reads configuration from environment variables (and .env file).
func Load() {
	// Load .env file if present (ignore error if not found)
	godotenv.Load()

	// Support MONGODB_URI, MONGO_URI, DATABASE_URL — in priority order
	mongoURI := getEnv("MONGODB_URI", "")
	if mongoURI == "" {
		mongoURI = getEnv("MONGO_URI", "")
	}
	if mongoURI == "" {
		mongoURI = getEnv("DATABASE_URL", "mongodb://localhost:27017")
	}

	AppConfig = Config{
		Port:        getEnv("PORT", "8087"),
		MongoURI:    mongoURI,
		StorageId:   getEnv("STORAGE_ID", ""),
		StoragePath: getEnv("STORAGE_PATH", ""),
		LogPath:     getEnv("LOG_PATH", "logs/server-spritesheet.log"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
