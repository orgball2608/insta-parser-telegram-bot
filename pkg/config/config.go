package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v9"
	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig       `envPrefix:"APP_"`
	Telegram  TelegramConfig  `envPrefix:"TELEGRAM_"`
	Instagram InstagramConfig `envPrefix:"INSTAGRAM_"`
	Postgres  PostgresConfig  `envPrefix:"POSTGRES_"`
	Redis     RedisConfig     `envPrefix:"REDIS_"`
}

type AppConfig struct {
	Environment string        `env:"ENV" envDefault:"development"`
	Port        int           `env:"PORT" envDefault:"8080"`
	LogLevel    string        `env:"LOG_LEVEL" envDefault:"info"`
	Timeout     time.Duration `env:"TIMEOUT" envDefault:"30s"`
	SentryUrl   string        `env:"SENTRY_URL" envDefault:""`
}

type TelegramConfig struct {
	BotToken string `env:"BOT_TOKEN,required"`
	User     int64  `env:"USER" envDefault:"0"`
	Channel  string `env:"CHANNEL" envDefault:""`
}

type InstagramConfig struct {
	UsersParse string `env:"USERS_PARSE" envDefault:""`
}

type PostgresConfig struct {
	Host     string `env:"HOST" envDefault:"localhost"`
	Port     int    `env:"PORT" envDefault:"5432"`
	User     string `env:"USER" envDefault:"postgres"`
	Pass     string `env:"PASSWORD" envDefault:"postgres"`
	Name     string `env:"NAME" envDefault:"insta_parser"`
	SslMode  string `env:"SSL_MODE" envDefault:"disable"`
	MaxConns int    `env:"MAX_CONNS" envDefault:"10"`
}

type RedisConfig struct {
	Host     string `env:"HOST" envDefault:"localhost"`
	Port     int    `env:"PORT" envDefault:"6379"`
	Password string `env:"PASSWORD" envDefault:""`
	DB       int    `env:"DB" envDefault:"0"`
}

func New() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: .env file not found: %v\n", err)
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Postgres.Host,
		c.Postgres.Port,
		c.Postgres.User,
		c.Postgres.Pass,
		c.Postgres.Name,
		c.Postgres.SslMode,
	)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}
