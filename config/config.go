package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppService      string `envconfig:"APP_SERVICE" default:"golang-base"`
	Port            string `envconfig:"PORT" default:"3100"`
	DatabaseURL     string `envconfig:"DATABASE_URL" required:"true"`
	RedisAddr       string `envconfig:"REDIS_ADDR"`
	RedisPassword   string `envconfig:"REDIS_PASSWORD"`
	TokenEncryptKey string `envconfig:"TOKEN_ENCRYPT_KEY"`
	RateLimitMax    int    `envconfig:"RATE_LIMIT_MAX" default:"100"`
	RateLimitExpMin int    `envconfig:"RATE_LIMIT_EXP_MINUTES" default:"1"`
	JWTSecret       string `envconfig:"JWT_SECRET" default:"default-secret-key-change-in-production"`

	// Logging
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	LogPretty bool   `envconfig:"LOG_PRETTY" default:"false"`

	// Observability
	AppVersion     string  `envconfig:"APP_VERSION" default:"dev"`
	AppEnvironment string  `envconfig:"APP_ENV" default:"development"`
	MetricsToken   string  `envconfig:"METRICS_TOKEN"`
	TracingEnabled bool    `envconfig:"TRACING_ENABLED" default:"false"`
	TracingSample  float64 `envconfig:"TRACING_SAMPLE_RATIO" default:"1"`

	// Proxy trust
	TrustedProxies []string `envconfig:"TRUSTED_PROXIES"`

	// SMTP Mailer
	SMTPHost     string `envconfig:"SMTP_HOST"`
	SMTPPort     int    `envconfig:"SMTP_PORT" default:"587"`
	SMTPUsername string `envconfig:"SMTP_USERNAME"`
	SMTPPassword string `envconfig:"SMTP_PASSWORD"`
	SMTPFrom     string `envconfig:"SMTP_FROM"`

	// Storage
	StorageProvider      string `envconfig:"STORAGE_PROVIDER" default:"local"`
	StorageLocalBasePath string `envconfig:"STORAGE_LOCAL_BASE_PATH" default:"./storage/app/public"`
	StoragePublicBaseURL string `envconfig:"STORAGE_PUBLIC_BASE_URL" default:"/storage"`
	MaxUploadSize        int64  `envconfig:"MAX_UPLOAD_SIZE" default:"10485760"` // 10MB default

	// S3 Configuration
	S3Bucket          string `envconfig:"S3_BUCKET"`
	S3Region          string `envconfig:"S3_REGION" default:"ap-southeast-1"`
	S3Endpoint        string `envconfig:"S3_ENDPOINT"`
	S3AccessKeyID     string `envconfig:"S3_ACCESS_KEY_ID"`
	S3SecretAccessKey string `envconfig:"S3_SECRET_ACCESS_KEY"`
	S3ForcePathStyle  bool   `envconfig:"S3_FORCE_PATH_STYLE" default:"false"`

	// GCS Configuration
	GCSBucket          string `envconfig:"GCS_BUCKET"`
	GCSCredentialsJSON string `envconfig:"GCS_CREDENTIALS_JSON"`
}

func findDotEnv() string {
	dir, err := os.Getwd()
	if err != nil {
		return ".env"
	}
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ".env"
}

func LoadConfig() *Config {
	envFile := findDotEnv()
	if err := godotenv.Load(envFile); err != nil && os.Getenv("DATABASE_URL") == "" {
		log.Println("Warning: No .env file found, using system environment variables")
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	// App Name / Service Fallback
	if cfg.AppService == "golang-base" {
		if appName := os.Getenv("APP_NAME"); appName != "" {
			cfg.AppService = appName
		}
	}

	// JWT Secret Fallback
	if cfg.JWTSecret == "" || cfg.JWTSecret == "default-secret-key-change-in-production" {
		if secret := os.Getenv("JWT_SECRET"); secret != "" {
			cfg.JWTSecret = secret
		}
	}

	// SMTP Fallbacks
	if cfg.SMTPUsername == "" {
		cfg.SMTPUsername = os.Getenv("SMTP_USER")
	}
	if cfg.SMTPPassword == "" {
		cfg.SMTPPassword = os.Getenv("SMTP_PASS")
	}

	// S3 / AWS Fallbacks
	if cfg.S3AccessKeyID == "" {
		cfg.S3AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if cfg.S3SecretAccessKey == "" {
		cfg.S3SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	if cfg.S3Region == "" {
		if r := os.Getenv("AWS_REGION"); r != "" {
			cfg.S3Region = r
		} else if r := os.Getenv("AWS_DEFAULT_REGION"); r != "" {
			cfg.S3Region = r
		}
	}
	if cfg.S3Bucket == "" {
		cfg.S3Bucket = os.Getenv("AWS_S3_BUCKET")
	}

	return &cfg
}
