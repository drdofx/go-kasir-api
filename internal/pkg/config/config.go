package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv              string
	Port                string
	DBConn              string
	JWTSecret           string
	AdminPassword       string
	CORSAllowedOrigin   string
	LogLevel            string
	MigrationsPath      string
	AutoMigrate         bool
	ServerReadTimeout   time.Duration
	ReadHeaderTimeout   time.Duration
	ServerWriteTimeout  time.Duration
	ServerIdleTimeout   time.Duration
	ServerShutdown      time.Duration
	MaxRequestBodyBytes int64
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
}

func Load() (Config, error) {
	v := viper.New()
	v.SetDefault("PORT", "8080")
	v.SetDefault("CORS_ALLOWED_ORIGIN", "http://localhost:8080")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("MIGRATIONS_PATH", "migrations")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("AUTO_MIGRATE", true)
	v.SetDefault("SERVER_READ_TIMEOUT", "10s")
	v.SetDefault("SERVER_READ_HEADER_TIMEOUT", "5s")
	v.SetDefault("SERVER_WRITE_TIMEOUT", "30s")
	v.SetDefault("SERVER_IDLE_TIMEOUT", "120s")
	v.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "10s")
	v.SetDefault("MAX_REQUEST_BODY_BYTES", 1048576)
	v.SetDefault("ACCESS_TOKEN_TTL", "15m")
	v.SetDefault("REFRESH_TOKEN_TTL", "720h")
	v.AutomaticEnv()
	v.SetConfigFile(".env")
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !os.IsNotExist(err) && !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := Config{
		AppEnv:              strings.ToLower(strings.TrimSpace(v.GetString("APP_ENV"))),
		Port:                v.GetString("PORT"),
		DBConn:              strings.TrimSpace(v.GetString("DB_CONN")),
		JWTSecret:           strings.TrimSpace(v.GetString("JWT_SECRET")),
		AdminPassword:       strings.TrimSpace(v.GetString("ADMIN_PASSWORD")),
		CORSAllowedOrigin:   v.GetString("CORS_ALLOWED_ORIGIN"),
		LogLevel:            v.GetString("LOG_LEVEL"),
		MigrationsPath:      v.GetString("MIGRATIONS_PATH"),
		AutoMigrate:         v.GetBool("AUTO_MIGRATE"),
		MaxRequestBodyBytes: v.GetInt64("MAX_REQUEST_BODY_BYTES"),
	}

	var err error
	if cfg.ServerReadTimeout, err = time.ParseDuration(v.GetString("SERVER_READ_TIMEOUT")); err != nil {
		return Config{}, fmt.Errorf("SERVER_READ_TIMEOUT must be a valid duration: %w", err)
	}
	if cfg.ReadHeaderTimeout, err = time.ParseDuration(v.GetString("SERVER_READ_HEADER_TIMEOUT")); err != nil {
		return Config{}, fmt.Errorf("SERVER_READ_HEADER_TIMEOUT must be a valid duration: %w", err)
	}
	if cfg.ServerWriteTimeout, err = time.ParseDuration(v.GetString("SERVER_WRITE_TIMEOUT")); err != nil {
		return Config{}, fmt.Errorf("SERVER_WRITE_TIMEOUT must be a valid duration: %w", err)
	}
	if cfg.ServerIdleTimeout, err = time.ParseDuration(v.GetString("SERVER_IDLE_TIMEOUT")); err != nil {
		return Config{}, fmt.Errorf("SERVER_IDLE_TIMEOUT must be a valid duration: %w", err)
	}
	if cfg.ServerShutdown, err = time.ParseDuration(v.GetString("SERVER_SHUTDOWN_TIMEOUT")); err != nil {
		return Config{}, fmt.Errorf("SERVER_SHUTDOWN_TIMEOUT must be a valid duration: %w", err)
	}
	if cfg.AccessTokenTTL, err = time.ParseDuration(v.GetString("ACCESS_TOKEN_TTL")); err != nil {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL must be a valid duration: %w", err)
	}
	if cfg.RefreshTokenTTL, err = time.ParseDuration(v.GetString("REFRESH_TOKEN_TTL")); err != nil {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL must be a valid duration: %w", err)
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if c.DBConn == "" {
		return fmt.Errorf("DB_CONN is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.MaxRequestBodyBytes < 0 {
		return fmt.Errorf("MAX_REQUEST_BODY_BYTES must be greater than or equal to 0")
	}
	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be greater than 0")
	}
	if c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be greater than 0")
	}
	if c.AppEnv == "production" {
		if len(c.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
		}
		if c.JWTSecret == "change-this-secret-in-production" {
			return fmt.Errorf("JWT_SECRET must not use the default development value in production")
		}
		if c.AdminPassword == "" {
			return fmt.Errorf("ADMIN_PASSWORD is required in production")
		}
	}
	return nil
}
