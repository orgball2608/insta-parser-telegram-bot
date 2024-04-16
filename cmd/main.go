package main

import (
	"log"
	"strings"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/config"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/db"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
)

func main() {
	cfg := config.GetConfig()

	bot, err := telegram.NewBot(cfg.Telegram.Token)
	if err != nil {
		log.Fatalln("Telegram bot create error:", err)
	}

	bot.SendError(cfg.Telegram.User, "bot start")

	pg, err := db.NewConnect(cfg)
	if err != nil {
		errString := err.Error()
		bot.SendError(cfg.Telegram.User, "BD error:"+errString)
	}

	err = pg.MigrationInit()
	if err != nil {
		errString := err.Error()
		bot.SendError(cfg.Telegram.User, "Migration error:"+errString)
	}

	insta := instagram.NewUser(cfg.Instagram.User, cfg.Instagram.Pass)
	if err = insta.LoginInstagram(cfg); err != nil {
		errString := err.Error()
		bot.SendError(cfg.Telegram.User, "Instagram login error:"+errString)
	}

	for {
		currentTime := getCurrentTime()
		hour := currentTime.Hour()
		if 12 <= hour && hour <= 24 {
			usernames := strings.Split(cfg.Instagram.UserParse, ";")
			for _, username := range usernames {
				err := parser.Start(insta, bot, pg, cfg, username)
				if err != nil {
					errString := err.Error()
					bot.SendError(cfg.Telegram.User, "Parser error:"+errString)
				}
			}
		}
		time.Sleep(time.Minute * time.Duration(cfg.Parser.Minutes))
	}
}

func getCurrentTime() time.Time {
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	return now.In(loc)
}
