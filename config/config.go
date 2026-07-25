package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port            string `envconfig:"PORT" default:"3100"`
	DatabaseURL     string `envconfig:"DATABASE_URL" required:"true"`
	RedisAddr       string `envconfig:"REDIS_ADDR"`
	RedisPassword   string `envconfig:"REDIS_PASSWORD"`
	TokenEncryptKey string `envconfig:"TOKEN_ENCRYPT_KEY"`
	RateLimitMax    int    `envconfig:"RATE_LIMIT_MAX" default:"100"`
	RateLimitExpMin int    `envconfig:"RATE_LIMIT_EXP_MINUTES" default:"1"`

	// SMTP Mailer
	SMTPHost     string `envconfig:"SMTP_HOST"`
	SMTPPort     int    `envconfig:"SMTP_PORT" default:"587"`
	SMTPUsername string `envconfig:"SMTP_USERNAME"`
	SMTPPassword string `envconfig:"SMTP_PASSWORD"`
	SMTPFrom     string `envconfig:"SMTP_FROM"`
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, using system environment variables")
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	return &cfg
}
