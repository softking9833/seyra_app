package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	SessionPepper string
}

func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func FromEnv() (Config, error) {
	cfg := Config{
		HTTPAddr:      getenv("SEYRA_HTTP_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("SEYRA_DATABASE_URL"),
		SessionPepper: os.Getenv("SEYRA_SESSION_PEPPER"),
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return Config{}, fmt.Errorf("SEYRA_DATABASE_URL is required")
	}
	if len(cfg.SessionPepper) < 16 {
		return Config{}, fmt.Errorf("SEYRA_SESSION_PEPPER must be at least 16 characters")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
