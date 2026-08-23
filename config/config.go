package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath    string
	ImageDir  string
	CredsFile string
	TokenFile string
	ProjectID string
	TopicName string
	SubName   string
	HTTPAddr  string
}

func Load() (Config, error) {
	_ = godotenv.Load() // .env is optional in production

	cfg := Config{
		DBPath:    os.Getenv("DB_PATH"),
		ImageDir:  os.Getenv("IMAGE_PATH"),
		CredsFile: envOr("CREDS_FILE", "credentials.json"),
		TokenFile: envOr("TOKEN_FILE", "token.json"),
		ProjectID: os.Getenv("GCloudProjectID"),
		TopicName: os.Getenv("GCloudTopicName"),
		SubName:   os.Getenv("GCloudGmailSubscription"),
		HTTPAddr:  envOr("HTTP_ADDR", "127.0.0.1:8080"),
	}

	if cfg.DBPath == "" {
		return cfg, fmt.Errorf("DB_PATH must be set")
	}
	if cfg.ImageDir == "" {
		return cfg, fmt.Errorf("IMAGE_PATH must be set")
	}
	if cfg.ProjectID == "" {
		return cfg, fmt.Errorf("GCloudProjectID must be set")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
