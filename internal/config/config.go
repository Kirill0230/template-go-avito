package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTP     HTTPConfig
	LogLevel string

	DatabaseURL             string
	DatabaseMaxConns        int
	DatabaseMinConns        int
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

type HTTPConfig struct {
	Addr              string
	ShutdownTimeout   time.Duration
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()

	var (
		cfg Config
		err error
	)

	cfg.HTTP.Addr, err = requireString("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	cfg.LogLevel, err = requireString("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.ShutdownTimeout, err = requireDuration("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseURL, err = requireString("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMaxConns, err = requireInt("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMinConns, err = requireInt("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}

	if cfg.DatabaseMaxConns <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than zero")
	}
	if cfg.DatabaseMinConns < 0 {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must not be negative")
	}
	if cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}

	cfg.DatabaseMaxConnLifetime, err = requireDuration("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseConnectTimeout, err = requireDuration("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseQueryTimeout, err = requireDuration("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.ReadTimeout, err = requireDuration("HTTP_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.ReadHeaderTimeout, err = requireDuration("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.WriteTimeout, err = requireDuration("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.IdleTimeout, err = requireDuration("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func requireString(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

func requireInt(key string) (int, error) {
	v, err := requireString(key)
	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return n, nil
}

func requireDuration(key string) (time.Duration, error) {
	v, err := requireString(key)
	if err != nil {
		return 0, err
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}

	return d, nil
}
