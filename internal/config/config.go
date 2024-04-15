package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Postgres struct {
		Port    int    `env:"POSTGRES_PORT"`
		Host    string `env:"POSTGRES_HOST"`
		User    string `env:"POSTGRES_USER"`
		Pass    string `env:"POSTGRES_PASS"`
		Name    string `env:"POSTGRES_NAME"`
		SslMode string `env:"POSTGRES_SSL_MODE"`
	}
	Telegram struct {
		User    int64  `env:"TELEGRAM_USER"`
		Token   string `env:"TELEGRAM_TOKEN"`
		Channel string `env:"TELEGRAM_CHANNEL"`
	}
	Instagram struct {
		User      string `env:"INSTAGRAM_USER"`
		Pass      string `env:"INSTAGRAM_PASS"`
		UserParse string `env:"INSTAGRAM_USER_PARSE"`
	}
	Parser struct {
		Minutes int `env:"PARSER_MINUTES"`
	}
}

var (
	once sync.Once
	cfg  *Config
)

func GetConfig() *Config {
	once.Do(func() {
		cfg = &Config{}
		if err := cleanenv.ReadConfig(".env", cfg); err != nil {
			help, _ := cleanenv.GetDescription(cfg, nil)
			log.Fatalln(err, help)
		}

	})
	return cfg
}
