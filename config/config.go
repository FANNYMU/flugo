package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	JWT      JWTConfig      `json:"jwt"`
	Upload   UploadConfig   `json:"upload"`
	Logger   LoggerConfig   `json:"logger"`
	Email    EmailConfig    `json:"email"`
	Queue    QueueConfig    `json:"queue"`
}

type ServerConfig struct {
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	ReadTimeout     int      `json:"read_timeout"`
	WriteTimeout    int      `json:"write_timeout"`
	AllowedOrigins  []string `json:"allowed_origins"`
	MaxRequestSize  int64    `json:"max_request_size"`
	EnableSwagger   bool     `json:"enable_swagger"`
	EnableMetrics   bool     `json:"enable_metrics"`
	EnableProfiling bool     `json:"enable_profiling"`
}

type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"`
	MaxIdle  int    `json:"max_idle"`
	MaxOpen  int    `json:"max_open"`
}

type EmailConfig struct {
	SMTPHost   string `json:"smtp_host"`
	SMTPPort   int    `json:"smtp_port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"from_email"`
	FromName   string `json:"from_name"`
	ReplyTo    string `json:"reply_to"`
	EnableSSL  bool   `json:"enable_ssl"`
	EnableAuth bool   `json:"enable_auth"`
}

type QueueConfig struct {
	Workers    int  `json:"workers"`
	BufferSize int  `json:"buffer_size"`
	Enabled    bool `json:"enabled"`
}

type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	Database int    `json:"database"`
}

type JWTConfig struct {
	Secret         string `json:"secret"`
	ExpirationTime int    `json:"expiration_time"`
	RefreshTime    int    `json:"refresh_time"`
}

type UploadConfig struct {
	MaxFileSize   int64    `json:"max_file_size"`
	AllowedTypes  []string `json:"allowed_types"`
	UploadPath    string   `json:"upload_path"`
	EnableResize  bool     `json:"enable_resize"`
	ThumbnailSize int      `json:"thumbnail_size"`
}

type LoggerConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputFile string `json:"output_file"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
}

var AppConfig *Config

func Load() *Config {
	config := &Config{
		Server: ServerConfig{
			Port:            getEnvInt("SERVER_PORT", 8080),
			Host:            getEnvString("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:     getEnvInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout:    getEnvInt("SERVER_WRITE_TIMEOUT", 30),
			AllowedOrigins:  getEnvStringSlice("SERVER_ALLOWED_ORIGINS", []string{"*"}),
			MaxRequestSize:  getEnvInt64("SERVER_MAX_REQUEST_SIZE", 10*1024*1024),
			EnableSwagger:   getEnvBool("SERVER_ENABLE_SWAGGER", true),
			EnableMetrics:   getEnvBool("SERVER_ENABLE_METRICS", true),
			EnableProfiling: getEnvBool("SERVER_ENABLE_PROFILING", false),
		},
		Database: DatabaseConfig{
			Driver:   getEnvString("DB_DRIVER", "sqlite3"),
			Host:     getEnvString("DB_HOST", ""),
			Port:     getEnvInt("DB_PORT", 0),
			Username: getEnvString("DB_USERNAME", ""),
			Password: getEnvString("DB_PASSWORD", ""),
			Database: getEnvString("DB_DATABASE", "storage/database.db"),
			SSLMode:  getEnvString("DB_SSL_MODE", ""),
			MaxIdle:  getEnvInt("DB_MAX_IDLE", 10),
			MaxOpen:  getEnvInt("DB_MAX_OPEN", 100),
		},
		Redis: RedisConfig{
			Host:     getEnvString("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnvString("REDIS_PASSWORD", ""),
			Database: getEnvInt("REDIS_DATABASE", 0),
		},
		JWT: JWTConfig{
			Secret:         getEnvString("JWT_SECRET", "flugo-secret-key"),
			ExpirationTime: getEnvInt("JWT_EXPIRATION_TIME", 3600),
			RefreshTime:    getEnvInt("JWT_REFRESH_TIME", 86400),
		},
		Upload: UploadConfig{
			MaxFileSize:   getEnvInt64("UPLOAD_MAX_FILE_SIZE", 10*1024*1024),
			AllowedTypes:  getEnvStringSlice("UPLOAD_ALLOWED_TYPES", []string{"image/jpeg", "image/png", "image/gif"}),
			UploadPath:    getEnvString("UPLOAD_PATH", "./uploads"),
			EnableResize:  getEnvBool("UPLOAD_ENABLE_RESIZE", true),
			ThumbnailSize: getEnvInt("UPLOAD_THUMBNAIL_SIZE", 200),
		},
		Logger: LoggerConfig{
			Level:      getEnvString("LOG_LEVEL", "info"),
			Format:     getEnvString("LOG_FORMAT", "json"),
			OutputFile: getEnvString("LOG_OUTPUT_FILE", ""),
			MaxSize:    getEnvInt("LOG_MAX_SIZE", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 3),
			MaxAge:     getEnvInt("LOG_MAX_AGE", 28),
		},
		Email: EmailConfig{
			SMTPHost:   getEnvString("EMAIL_SMTP_HOST", "localhost"),
			SMTPPort:   getEnvInt("EMAIL_SMTP_PORT", 587),
			Username:   getEnvString("EMAIL_USERNAME", ""),
			Password:   getEnvString("EMAIL_PASSWORD", ""),
			FromEmail:  getEnvString("EMAIL_FROM_EMAIL", "noreply@example.com"),
			FromName:   getEnvString("EMAIL_FROM_NAME", "Flugo Framework"),
			ReplyTo:    getEnvString("EMAIL_REPLY_TO", ""),
			EnableSSL:  getEnvBool("EMAIL_ENABLE_SSL", true),
			EnableAuth: getEnvBool("EMAIL_ENABLE_AUTH", true),
		},
		Queue: QueueConfig{
			Workers:    getEnvInt("QUEUE_WORKERS", 5),
			BufferSize: getEnvInt("QUEUE_BUFFER_SIZE", 1000),
			Enabled:    getEnvBool("QUEUE_ENABLED", true),
		},
	}

	if configFile := getEnvString("CONFIG_FILE", ""); configFile != "" {
		loadFromFile(config, configFile)
	}

	AppConfig = config
	return config
}

func loadFromFile(config *Config, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(config)
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.Driver,
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.SSLMode,
	)
}
