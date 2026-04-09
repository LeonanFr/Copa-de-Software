package config

import (
	"os"
)

type Config struct {
	ServerPort string
	MongoURI   string
	MongoDB    string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = getEnv("SERVER_PORT", "8080")
	}

	return Config{
		ServerPort: port,
		MongoURI:   getEnv("MONGO_URI", ""),
		MongoDB:    getEnv("MONGO_DB", "copasoftware"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
