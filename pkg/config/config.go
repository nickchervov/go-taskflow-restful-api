package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	POSTGRES_DB       string `yaml:"POSTGRES_DB"`
	POSTGRES_USER     string `yaml:"POSTGRES_USER"`
	POSTGRES_PASSWORD string `yaml:"POSTGRES_PASSWORD"`
	DATABASE_URL      string `yaml:"DATABASE_URL"`
	REDIS_URL         string `yaml:"REDIS_URL"`
	WORKER_COUNT      int    `yaml:"WORKER_COUNT"`
	RATE_LIMIT        int    `yaml:"RATE_LIMIT"`
	LOG_LEVEL         string `yaml:"LOG_LEVEL"`
	LOG_FORMAT        string `yaml:"LOG_FORMAT"`
}

func NewConfig() *Config {
	return &Config{}
}

// Если запускать из Docker или с переменными окружения
func (c *Config) Load() error {
	c.POSTGRES_DB = os.Getenv("POSTGRES_DB")
	c.POSTGRES_USER = os.Getenv("POSTGRES_USER")
	c.POSTGRES_PASSWORD = os.Getenv("POSTGRES_PASSWORD")
	c.DATABASE_URL = os.Getenv("DATABASE_URL")
	c.REDIS_URL = os.Getenv("REDIS_URL")
	count, err := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	if err != nil {
		return fmt.Errorf("converting workers count environment to int: %w", err)
	}
	c.WORKER_COUNT = count
	limit, err := strconv.Atoi(os.Getenv("RATE_LIMIT"))
	if err != nil {
		return fmt.Errorf("converting rate limit environment to int: %w", err)
	}
	c.RATE_LIMIT = limit

	c.LOG_LEVEL = os.Getenv("LOG_LEVEL")
	c.LOG_FORMAT = os.Getenv("LOG_FORMAT")
	return nil
}

// // Чтение конфига просто из файла
// func (c *Config) Load() error {
// 	if err := cleanenv.ReadConfig("./pkg/config/config.yaml", c); err != nil {
// 		return fmt.Errorf("load config: %w", err)
// 	}
// 	return nil
// }
