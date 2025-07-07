package commandimpl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/formatter"
)

func (c *CommandImpl) handlePostCommand(ctx context.Context, update tgbotapi.Update) error {
	postURL := strings.TrimSpace(update.Message.CommandArguments())
	chatID := update.Message.Chat.ID

	if postURL == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a post URL: /post <instagram_post_url>")
		return err
	}

	escapedURL := formatter.EscapeMarkdownV2(postURL)
	initialMessage := fmt.Sprintf("Fetching post from URL: %s... ⏳", escapedURL)
	sentMsgID, err := c.Telegram.SendMessage(chatID, initialMessage)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var post *domain.PostItem

	op := func() error {
		var opErr error
		post, opErr = c.Instagram.GetUserPost(ctxWithTimeout, postURL)
		return opErr
	}

	err = c.doWithRetryNotify(ctx, chatID, sentMsgID, initialMessage, "GetUserPost", op)

	if err != nil {
		userFriendlyError := "Đã xảy ra lỗi khi cố gắng lấy nội dung từ Instagram. Vui lòng thử lại sau."

		if errors.Is(err, instagram.ErrPrivateAccount) {
			userFriendlyError = "Rất tiếc, tài khoản Instagram này là riêng tư và không thể truy cập được."
		} else if strings.Contains(err.Error(), "could not parse URL") || strings.Contains(err.Error(), "invalid URL") {
			userFriendlyError = "URL Instagram bạn cung cấp không hợp lệ. Vui lòng kiểm tra lại."
		} else if strings.Contains(err.Error(), "no download links found") || strings.Contains(err.Error(), "Could not find any media in the provided URL.") {
			userFriendlyError = "Không tìm thấy bất kỳ phương tiện nào trong URL bạn cung cấp. Vui lòng kiểm tra lại."
		} else if strings.Contains(err.Error(), "Timeout waiting for media result") {
			userFriendlyError = "Yêu cầu lấy nội dung từ Instagram đã hết thời gian. Vui lòng thử lại sau."
		}

		if editErr := c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("❌ Lỗi: %s", userFriendlyError)); editErr != nil {
			c.Logger.Error("Failed to edit message text with user-friendly error", "error", editErr)
		}
		return fmt.Errorf("failed to get post from URL: %w", err)
	}

	if len(post.MediaURLs) == 0 {
		if err := c.Telegram.EditMessageText(chatID, sentMsgID, "Không tìm thấy bất kỳ phương tiện nào trong URL bạn cung cấp."); err != nil {
			c.Logger.Error("Failed to edit message text", "error", err)
		}
		return nil
	}

	post.PostURL = postURL

	if err := c.Telegram.EditMessageText(chatID, sentMsgID, "✅ Successfully fetched post info! Sending media now..."); err != nil {
		c.Logger.Error("Failed to edit message text", "error", err)
	}

	mediaGroup := make([]interface{}, 0, len(post.MediaURLs))
	var captionBuilder strings.Builder
	if post.Username != "" {
		escapedUsername := formatter.EscapeMarkdownV2(post.Username)
		captionBuilder.WriteString(fmt.Sprintf("*Post by @%s*\n\n", escapedUsername))
	}
	if post.Caption != "" {
		escapedCaption := formatter.EscapeMarkdownV2(post.Caption)
		captionBuilder.WriteString(escapedCaption)
		captionBuilder.WriteString("\n\n")
	}
	if post.LikeCount > 0 {
		captionBuilder.WriteString(fmt.Sprintf("❤️ %s", formatter.FormatNumber(post.LikeCount)))
	}
	if post.PostedAgo != "" {
		escapedPostedAgo := formatter.EscapeMarkdownV2(post.PostedAgo)
		captionBuilder.WriteString(fmt.Sprintf(" | 🕒 %s\n", escapedPostedAgo))
	} else if post.LikeCount > 0 {
		captionBuilder.WriteString("\n")
	}

	captionBuilder.WriteString(fmt.Sprintf("\n[View on Instagram](%s)", post.PostURL))

	captionToSend := captionBuilder.String()

	for i, mediaURL := range post.MediaURLs {
		var mediaItem tgbotapi.RequestFileData = tgbotapi.FileURL(mediaURL)
		var media tgbotapi.BaseInputMedia

		if strings.Contains(mediaURL, ".mp4") {
			video := tgbotapi.NewInputMediaVideo(mediaItem)
			if i == 0 {
				video.Caption = captionToSend
			}
			media = video.BaseInputMedia
		} else {
			photo := tgbotapi.NewInputMediaPhoto(mediaItem)
			if i == 0 {
				photo.Caption = captionToSend
			}
			media = photo.BaseInputMedia
		}
		mediaGroup = append(mediaGroup, media)
	}

	if len(mediaGroup) > 0 {
		if err := c.Telegram.SendMediaGroup(chatID, mediaGroup); err != nil {
			c.Logger.Error("Failed to send media group, falling back to individual sending", "error", err)

			if captionToSend != "" {
				if _, err := c.Telegram.SendMessage(chatID, captionToSend); err != nil {
					c.Logger.Error("Failed to send message", "error", err)
				}
			}

			totalMedia := len(post.MediaURLs)
			for i, mediaURL := range post.MediaURLs {
				progressMessage := fmt.Sprintf("Đang gửi phương tiện %d/%d... 📤", i+1, totalMedia)
				if editErr := c.Telegram.EditMessageText(chatID, sentMsgID, progressMessage); editErr != nil {
					c.Logger.Error("Failed to edit message text with progress", "error", editErr)
				}

				if err := c.Telegram.SendMediaByUrl(chatID, mediaURL); err != nil {
					c.Logger.Error("Failed to send media by URL", "error", err)
				}
			}
			if editErr := c.Telegram.EditMessageText(chatID, sentMsgID, "✅ Đã gửi tất cả phương tiện."); editErr != nil {
				c.Logger.Error("Failed to edit message text after individual sending completion", "error", editErr)
			}
		} else {
			// If media group was sent successfully, update the message to indicate completion
			if editErr := c.Telegram.EditMessageText(chatID, sentMsgID, "✅ Đã gửi tất cả phương tiện."); editErr != nil {
				c.Logger.Error("Failed to edit message text after media group completion", "error", editErr)
			}
		}
	}

	return nil
}
