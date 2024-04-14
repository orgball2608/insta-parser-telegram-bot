package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Postgres struct {
		Port    int    `yaml:"port"`
		Host    string `yaml:"host"`
		User    string `yaml:"user"`
		Pass    string `yaml:"pass"`
		Name    string `yaml:"name"`
		Sslmode string `yaml:"sslmode"`
	}
	Telegram struct {
		User    int64  `yaml:"user"`
		Token   string `yaml:"token"`
		Channel string `yaml:"channel"`
	}
	Instagram struct {
		User      string `yaml:"username"`
		Pass      string `yaml:"password"`
		UserParse string `ymal:"userParse"`
	}
	Parser struct {
		Minutes int `yaml:"minutes"`
	}
}

var (
	once sync.Once
	cfg  *Config
)

func GetConfig() *Config {
	once.Do(func() {
		cfg = &Config{}
		if err := cleanenv.ReadConfig("config.yml", cfg); err != nil {
			help, _ := cleanenv.GetDescription(cfg, nil)
			log.Fatalln(err, help)
		}

	})
	return cfg
}
