package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App struct {
		Env       string `env:"APP_ENV" env-default:"development"`
		Port      int    `env:"APP_PORT" env-default:"8080"`
		SentryUrl string `env:"SENTRY_URL"`
	}
	Postgres struct {
		Port    int    `env:"POSTGRES_PORT"`
		Host    string `env:"POSTGRES_HOST"`
		User    string `env:"POSTGRES_USER"`
		Pass    string `env:"POSTGRES_PASS"`
		Name    string `env:"POSTGRES_NAME"`
		SslMode string `env:"POSTGRES_SSL_MODE" env-default:"disable"`
	}
	Telegram struct {
		User    int64  `env:"TELEGRAM_USER"`
		Token   string `env:"TELEGRAM_TOKEN"`
		Channel string `env:"TELEGRAM_CHANNEL"`
	}
	Instagram struct {
		User        string `env:"INSTAGRAM_USER"`
		Pass        string `env:"INSTAGRAM_PASS"`
		SessionPath string `env:"INSTAGRAM_SESSION_PATH" env-default:"./goinsta-session"`
		UsersParse  string `env:"INSTAGRAM_USERS_PARSE"`
	}
}

var (
	once sync.Once
	cfg  *Config
)

func New() (*Config, error) {
	once.Do(func() {
		cfg = &Config{}
		if err := cleanenv.ReadEnv(cfg); err != nil {
			help, _ := cleanenv.GetDescription(cfg, nil)
			log.Fatalf("Failed to read configuration: %v\n%v", err, help)
		}
	})
	return cfg, nil
}
