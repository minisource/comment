package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the comment service
type Config struct {
	Server     ServerConfig
	MongoDB    MongoDBConfig
	Redis      RedisConfig
	Auth       AuthConfig
	Notifier   NotifierConfig
	Moderation ModerationConfig
	Logging    LoggingConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// MongoDBConfig holds MongoDB configuration
type MongoDBConfig struct {
	URI             string
	Database        string
	MaxPoolSize     uint64
	MinPoolSize     uint64
	MaxConnIdleTime time.Duration
}

// RedisConfig holds Redis configuration for caching
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// AuthConfig holds auth service configuration
type AuthConfig struct {
	ServiceURL        string
	IntrospectionPath string
	ClientID          string
	ClientSecret      string
	CacheSeconds      int
	SkipPaths         []string
}

// NotifierConfig holds notifier service configuration
type NotifierConfig struct {
	ServiceURL   string
	ClientID     string
	ClientSecret string
	Enabled      bool
}

// ModerationConfig holds content moderation settings
type ModerationConfig struct {
	RequireApproval    bool
	BadWordsEnabled    bool
	BadWordsList       []string
	MaxCommentLength   int
	MaxReplyDepth      int
	AllowAnonymous     bool
	RateLimitPerMinute int
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port:            getEnvAsInt("SERVER_PORT", 5010),
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:     getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		MongoDB: MongoDBConfig{
			URI:             getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:        getEnv("MONGODB_DATABASE", "minisource_comments"),
			MaxPoolSize:     uint64(getEnvAsInt("MONGODB_MAX_POOL_SIZE", 100)),
			MinPoolSize:     uint64(getEnvAsInt("MONGODB_MIN_POOL_SIZE", 10)),
			MaxConnIdleTime: getDuration("MONGODB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 2),
		},
		Auth: AuthConfig{
			ServiceURL:        getEnv("AUTH_SERVICE_URL", "http://localhost:5001"),
			IntrospectionPath: getEnv("AUTH_INTROSPECTION_PATH", "/api/v1/oauth/introspect"),
			ClientID:          getEnv("AUTH_CLIENT_ID", "comment-service"),
			ClientSecret:      getEnv("AUTH_CLIENT_SECRET", "comment-service-secret-key"),
			CacheSeconds:      getEnvAsInt("AUTH_CACHE_SECONDS", 300),
			SkipPaths:         getEnvAsSlice("AUTH_SKIP_PATHS", []string{"/health", "/ready", "/metrics"}),
		},
		Notifier: NotifierConfig{
			ServiceURL:   getEnv("NOTIFIER_SERVICE_URL", "http://localhost:5003"),
			ClientID:     getEnv("NOTIFIER_CLIENT_ID", "comment-service"),
			ClientSecret: getEnv("NOTIFIER_CLIENT_SECRET", "comment-service-secret-key"),
			Enabled:      getEnvAsBool("NOTIFIER_ENABLED", true),
		},
		Moderation: ModerationConfig{
			RequireApproval:    getEnvAsBool("MODERATION_REQUIRE_APPROVAL", true),
			BadWordsEnabled:    getEnvAsBool("MODERATION_BAD_WORDS_ENABLED", true),
			BadWordsList:       getEnvAsSlice("MODERATION_BAD_WORDS", getDefaultBadWords()),
			MaxCommentLength:   getEnvAsInt("MODERATION_MAX_COMMENT_LENGTH", 5000),
			MaxReplyDepth:      getEnvAsInt("MODERATION_MAX_REPLY_DEPTH", 5),
			AllowAnonymous:     getEnvAsBool("MODERATION_ALLOW_ANONYMOUS", false),
			RateLimitPerMinute: getEnvAsInt("MODERATION_RATE_LIMIT_PER_MINUTE", 10),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}, nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getDefaultBadWords() []string {
	// This is a minimal list - in production, load from file or database
	return []string{
		"spam", "scam", "xxx", "porn",
	}
}
