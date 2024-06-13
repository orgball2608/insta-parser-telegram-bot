package commandimpl

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

func (c *CommandImpl) HandleCommand() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := c.Telegram.GetUpdatesChan(u)

	if err != nil {
		c.Logger.Error("Error getting updates from telegram", "Error", err)
		return err
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}
		c.Logger.Info("Message received", "From", update.Message.From.UserName, "Text", update.Message.Text)

		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "story":
				{
					msg.Text = "Getting stories by username"
					text := strings.ReplaceAll(update.Message.Text, "/story", "")
					userName := strings.TrimSpace(text)
					c.Logger.Info("Getting stories by username", "Username", text)
					highlights, err := c.Instagram.GetUserHighlights(userName)

					if err != nil {
						c.Logger.Error("Error getting highlights", "Error", err)
						return err
					}

					c.Logger.Info("Highlight", "Length of highlight", len(highlights))

					for _, highlight := range highlights {
						c.Logger.Info("Highlight", "Title", highlight.Title)
						if len(highlight.Items) == 0 {
							c.Logger.Info("Highlight", "No items")
							continue
						}
						err = c.Parser.ParseStories(highlight.Items)
						if err != nil {
							c.Logger.Error("Error parsing stories", "Error", err)
							return err
						}
					}

				}
			default:
				msg.Text = "I don't know that command"
			}
			c.Telegram.SendMessageToUser(msg.Text)
		}
	}

	return nil
}

func (c *CommandImpl) GetStoryByUserNameCommand() {
	panic("implement")
}
