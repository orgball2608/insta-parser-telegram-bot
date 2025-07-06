package commandimpl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/formatter"
)

const helpMessage = `👋 *Welcome to the Instagram Parser Bot!*

Here are the available commands:

*AUTOMATIC SUBSCRIPTIONS:*
/subscribe <username> - Subscribe to a user to get new stories automatically.
/unsubscribe <username> - Unsubscribe from a user.
/listsubscriptions - List all your current subscriptions.

*ONE-TIME DOWNLOADS:*
/story <username> - Fetch all current stories from a user.
/highlights <username> - Fetch all highlights from a user.
/post <post_url> - Download a post (photo/video/album) from its URL.
/reel <reel_url> - Download a Reel from its URL.

Type /help at any time to see this guide.`

func (c *CommandImpl) HandleCommand(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := c.Telegram.GetUpdatesChan(u)
	c.Logger.Info("Command handler started, listening for updates.")

	for {
		select {
		case <-ctx.Done():
			c.Logger.Info("Command handler shutting down.")
			c.Telegram.StopReceivingUpdates()
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				c.Logger.Warn("Telegram updates channel closed unexpectedly. Restarting handler...")
				return errors.New("telegram updates channel closed")
			}
			if update.CallbackQuery != nil {
				go c.handleCallback(ctx, update.CallbackQuery)
				continue
			}

			go func(u tgbotapi.Update) {
				defer func() {
					if r := recover(); r != nil {
						c.Logger.Error("Panic recovered while processing an update", "panic", r, "stack", string(debug.Stack()))
					}
				}()

				if u.Message == nil {
					return
				}

				c.Logger.Info("Message received", "from", u.Message.From.UserName, "text", u.Message.Text)

				if u.Message.IsCommand() {
					if err := c.processCommand(ctx, u); err != nil {
						c.Logger.Error("Error processing command",
							"command", u.Message.Command(),
							"error", err)
					}
				}
			}(update)
		}
	}
}

func (c *CommandImpl) processCommand(ctx context.Context, update tgbotapi.Update) error {
	command := update.Message.Command()
	args := update.Message.CommandArguments()
	chatID := update.Message.Chat.ID

	switch command {
	case "start", "help":
		_, err := c.Telegram.SendMessage(chatID, helpMessage)
		return err
	case "subscribe", "unsubscribe", "listsubscriptions":
		switch command {
		case "subscribe":
			c.handleSubscribe(ctx, chatID, args)
		case "unsubscribe":
			c.handleUnsubscribe(ctx, chatID, args)
		case "listsubscriptions":
			c.handleListSubscriptions(ctx, chatID)
		}
		return nil
	}

	if !c.RateLimiter.Allow(chatID) {
		if _, err := c.Telegram.SendMessage(chatID, "⏳ You are making requests too quickly. Please wait a moment and try again."); err != nil {
			c.Logger.Error("Failed to send message", "error", err)
		}
		return nil
	}

	switch command {
	case "story":
		return c.handleStoryCommand(ctx, update)
	case "highlights":
		return c.handleHighlightsCommand(ctx, update)
	case "post":
		return c.handlePostCommand(ctx, update)
	case "reel":
		return c.handleReelCommand(ctx, update)
	default:
		_, err := c.Telegram.SendMessage(chatID, "Unknown command. Type /help to see the list of available commands.")
		return err
	}
}

func (c *CommandImpl) handleStoryCommand(ctx context.Context, update tgbotapi.Update) error {
	args := strings.TrimSpace(update.Message.CommandArguments())
	userName := strings.TrimSpace(args)
	chatID := update.Message.Chat.ID

	if userName == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a username: /story <username>")
		return err
	}

	escapedUser := formatter.EscapeMarkdownV2(userName)
	initialMessage := fmt.Sprintf("Fetching stories for @%s... ⏳", escapedUser)
	sentMsgID, err := c.Telegram.SendMessage(chatID, initialMessage)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	var stories []domain.StoryItem
	op := func() error {
		var opErr error
		stories, opErr = c.Instagram.GetUserStories(userName)
		return opErr
	}

	err = c.doWithRetryNotify(ctx, chatID, sentMsgID, initialMessage, "GetUserStories", op)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Error fetching stories for @%s: %v", escapedUser, err)
		if errors.Is(err, instagram.ErrPrivateAccount) {
			errMsg = fmt.Sprintf("Account @%s is private, I cannot fetch stories.", escapedUser)
		}
		if err := c.Telegram.EditMessageText(chatID, sentMsgID, errMsg); err != nil {
			c.Logger.Error("Failed to edit message text", "error", err)
		}
		return err
	}

	if len(stories) == 0 {
		if err := c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("No current stories found for @%s.", escapedUser)); err != nil {
			c.Logger.Error("Failed to edit message text", "error", err)
		}
		return nil
	}

	if err := c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("✅ Found %d stories for @%s. Sending now...", len(stories), escapedUser)); err != nil {
		c.Logger.Error("Failed to edit message text", "error", err)
	}

	if err := c.Parser.ClearCurrentStories(userName); err != nil {
		c.Logger.Error("Error clearing current stories", "error", err)
	}

	for _, item := range stories {
		if item.MediaURL == "" {
			continue
		}
		if err := c.Telegram.SendMediaByUrl(chatID, item.MediaURL); err != nil {
			c.Logger.Error("Failed to send story media", "url", item.MediaURL, "error", err)
		}
	}

	if _, err := c.Telegram.SendMessage(chatID, fmt.Sprintf("Finished sending %d stories for @%s.", len(stories), escapedUser)); err != nil {
		c.Logger.Error("Failed to send message", "error", err)
	}
	return nil
}

func (c *CommandImpl) handleHighlightsCommand(ctx context.Context, update tgbotapi.Update) error {
	userName := strings.TrimSpace(update.Message.CommandArguments())
	chatID := update.Message.Chat.ID

	if userName == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a username: /hls <username>")
		return err
	}

	escapedUser := formatter.EscapeMarkdownV2(userName)
	initialMessage := fmt.Sprintf("Fetching highlight albums for @%s... ⏳", escapedUser)
	sentMsgID, err := c.Telegram.SendMessage(chatID, initialMessage)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	var previews []domain.HighlightAlbumPreview
	op := func() error {
		var opErr error
		previews, opErr = c.Instagram.GetHighlightAlbumPreviews(userName)
		return opErr
	}

	err = c.doWithRetryNotify(ctx, chatID, sentMsgID, initialMessage, "GetHighlightAlbumPreviews", op)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Error fetching highlights for @%s: %v", escapedUser, err)
		if errors.Is(err, instagram.ErrPrivateAccount) {
			errMsg = fmt.Sprintf("Account @%s is private, I cannot fetch highlights.", escapedUser)
		}
		if err := c.Telegram.EditMessageText(chatID, sentMsgID, errMsg); err != nil {
			c.Logger.Error("Failed to edit message text", "error", err)
		}
		return err
	}

	if len(previews) == 0 {
		if err := c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("No highlights found for @%s.", escapedUser)); err != nil {
			c.Logger.Error("Failed to edit message text", "error", err)
		}
		return nil
	}

	var keyboardRows [][]tgbotapi.InlineKeyboardButton
	for _, preview := range previews {
		callbackData, _ := json.Marshal(map[string]string{
			"action":   "dl_highlight",
			"user":     userName,
			"album_id": preview.ID,
		})

		button := tgbotapi.NewInlineKeyboardButtonData(preview.Title, string(callbackData))
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(button))
	}

	msgText := fmt.Sprintf("Found %d highlight albums for @%s\nPlease select an album to download:", len(previews), escapedUser)
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	if err := c.Telegram.DeleteMessage(tgbotapi.NewDeleteMessage(chatID, sentMsgID)); err != nil {
		c.Logger.Error("Failed to delete message", "error", err)
	}
	if _, err := c.Telegram.Send(msg); err != nil {
		c.Logger.Error("Failed to send message", "error", err)
	}

	return nil
}

// New method to handle callback queries from button clicks
func (c *CommandImpl) handleCallback(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) {
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	_, _ = c.Telegram.Request(callback)

	var callbackData struct {
		Action  string `json:"action"`
		User    string `json:"user"`
		AlbumID string `json:"album_id"`
	}

	if err := json.Unmarshal([]byte(callbackQuery.Data), &callbackData); err != nil {
		c.Logger.Error("Failed to unmarshal callback data", "error", err)
		return
	}

	chatID := callbackQuery.Message.Chat.ID

	switch callbackData.Action {
	case "dl_highlight":
		escapedUser := formatter.EscapeMarkdownV2(callbackData.User)
		c.Telegram.EditMessageText(
			chatID,
			callbackQuery.Message.MessageID,
			fmt.Sprintf("Downloading highlight album for @%s... ⏳", escapedUser),
		)

		c.downloadSingleHighlightAlbum(ctx, chatID, callbackData.User, callbackData.AlbumID, callbackQuery.Message.MessageID)
	}
}

// New method to download a single highlight album
func (c *CommandImpl) downloadSingleHighlightAlbum(ctx context.Context, chatID int64, userName, albumID string, messageID int) {
	highlightReel, err := c.Instagram.GetSingleHighlightAlbum(userName, albumID)
	if err != nil {
		escapedUser := formatter.EscapeMarkdownV2(userName)
		errMsg := fmt.Sprintf("❌ Error fetching highlight album for @%s: %v", escapedUser, err)
		if errors.Is(err, instagram.ErrPrivateAccount) {
			errMsg = fmt.Sprintf("Account @%s is private, I cannot fetch highlights.", escapedUser)
		}
		c.Telegram.EditMessageText(chatID, messageID, errMsg)
		return
	}

	if highlightReel == nil || len(highlightReel.Items) == 0 {
		c.Telegram.EditMessageText(chatID, messageID, "No items found in this highlight album.")
		return
	}

	var validItems []domain.StoryItem
	for _, item := range highlightReel.Items {
		if item.MediaURL != "" {
			validItems = append(validItems, item)
		}
	}

	if len(validItems) == 0 {
		c.Telegram.EditMessageText(chatID, messageID, "No valid media found in this highlight album.")
		return
	}

	escapedTitle := formatter.EscapeMarkdownV2(highlightReel.Title)
	totalItems := len(validItems)

	c.Telegram.EditMessageText(
		chatID,
		messageID,
		fmt.Sprintf("Found %d items in '%s'. Processing in batches...", totalItems, escapedTitle),
	)

	const batchSize = 10
	totalBatches := (totalItems + batchSize - 1) / batchSize

	for _, item := range validItems {
		highlightItem := domain.Highlights{
			UserName:  userName,
			MediaURL:  item.MediaURL,
			CreatedAt: time.Now(),
		}
		if err := c.Parser.SaveHighlight(highlightItem); err != nil {
			c.Logger.Error("Error saving highlight to DB", "url", item.MediaURL, "error", err)
		}
	}

	var successCount int
	for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
		startIdx := batchIndex * batchSize
		endIdx := startIdx + batchSize
		if endIdx > totalItems {
			endIdx = totalItems
		}

		batchItems := validItems[startIdx:endIdx]
		batchSize := len(batchItems)

		c.Telegram.EditMessageText(
			chatID,
			messageID,
			fmt.Sprintf("Processing '%s': Batch %d/%d (%d items)...",
				escapedTitle, batchIndex+1, totalBatches, batchSize),
		)

		batchSuccess := c.processBatch(ctx, chatID, batchItems, highlightReel.Title, batchIndex == 0)
		if batchSuccess {
			successCount += batchSize
		}
	}

	c.Telegram.EditMessageText(
		chatID,
		messageID,
		fmt.Sprintf("✅ Finished! Successfully sent %d/%d items from highlight album '%s'.",
			successCount, totalItems, escapedTitle),
	)
}

// processBatch handles downloading and sending a batch of media items
func (c *CommandImpl) processBatch(ctx context.Context, chatID int64, batchItems []domain.StoryItem, albumTitle string, isFirstBatch bool) bool {
	var wg sync.WaitGroup
	tempFilePathsChannel := make(chan string, len(batchItems))

	for _, item := range batchItems {
		wg.Add(1)
		go func(mediaItem domain.StoryItem) {
			defer wg.Done()

			filePath, err := c.Telegram.DownloadMediaToTempFile(mediaItem.MediaURL)
			if err != nil {
				c.Logger.Error("Failed to download media to temp file", "url", mediaItem.MediaURL, "error", err)
				return
			}
			tempFilePathsChannel <- filePath
		}(item)
	}

	wg.Wait()
	close(tempFilePathsChannel)

	var tempFilePaths []string
	for path := range tempFilePathsChannel {
		tempFilePaths = append(tempFilePaths, path)
	}

	defer func() {
		for _, path := range tempFilePaths {
			if err := os.Remove(path); err != nil {
				c.Logger.Warn("Failed to remove temp file", "path", path, "error", err)
			}
		}
	}()

	if len(tempFilePaths) == 0 {
		c.Logger.Warn("Failed to download any media from batch", "batch_size", len(batchItems))
		return false
	}

	mediaGroup := make([]interface{}, 0, len(tempFilePaths))
	for i, path := range tempFilePaths {
		var isVideo bool
		if i < len(batchItems) {
			isVideo = strings.Contains(batchItems[i].MediaURL, ".mp4")
		}

		fileData := tgbotapi.FilePath(path)

		if isVideo {
			mediaGroup = append(mediaGroup, tgbotapi.NewInputMediaVideo(fileData))
		} else {
			mediaGroup = append(mediaGroup, tgbotapi.NewInputMediaPhoto(fileData))
		}
	}

	if isFirstBatch && len(mediaGroup) > 0 {
		caption := fmt.Sprintf("Highlight: %s", albumTitle)
		switch m := mediaGroup[0].(type) {
		case tgbotapi.InputMediaVideo:
			m.Caption = caption
			mediaGroup[0] = m
		case tgbotapi.InputMediaPhoto:
			m.Caption = caption
			mediaGroup[0] = m
		}
	}

	if err := c.Telegram.SendMediaGroup(chatID, mediaGroup); err != nil {
		c.Logger.Error("Failed to send highlight media group batch", "title", albumTitle, "error", err)

		caption := fmt.Sprintf("Highlight: %s", albumTitle)
		if isFirstBatch {
			c.Telegram.SendMessage(chatID, caption)
		}

		for _, item := range batchItems {
			if err := c.Telegram.SendMediaByUrl(chatID, item.MediaURL); err != nil {
				c.Logger.Error("Failed to send individual media", "url", item.MediaURL, "error", err)
			}
		}

		return false
	}

	return true
}
