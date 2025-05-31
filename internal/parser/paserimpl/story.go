package paserimpl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Davincible/goinsta/v3"
	"github.com/go-co-op/gocron/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	storyRepo "github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
)

// ScheduleParseStories lên lịch phân tích stories của người dùng
func (p *ParserImpl) ScheduleParseStories(ctx context.Context) error {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.Local
		p.Logger.Warn("Failed to load Asia/Ho_Chi_Minh timezone, using local timezone", "error", err)
	}

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	// Lên lịch nhiệm vụ với khoảng thời gian ngẫu nhiên
	_, err = scheduler.NewJob(
		gocron.DurationRandomJob(
			time.Hour*1,
			time.Hour*24,
		),
		gocron.NewTask(
			func() {
				// Kiểm tra xem context đã bị hủy chưa
				if ctx.Err() != nil {
					p.Logger.Info("Context cancelled, stopping story parsing schedule")
					return
				}

				// Tạo context con với timeout
				taskCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
				defer cancel()

				usernames := strings.Split(p.Config.Instagram.UsersParse, ";")
				p.Logger.Info("Starting scheduled story parsing", "userCount", len(usernames))

				// Xử lý từng username
				for _, username := range usernames {
					username = strings.TrimSpace(username)
					if username == "" {
						continue
					}

					p.Logger.Info("Parsing stories for user", "username", username)
					if err := p.ParseUserReelStories(taskCtx, username); err != nil {
						p.Logger.Error("Failed to parse stories for user",
							"username", username,
							"error", err)
						p.Telegram.SendMessageToUser(fmt.Sprintf("Failed to parse stories for %s: %s",
							username, err.Error()))
					} else {
						p.Logger.Info("Successfully parsed stories for user", "username", username)
					}

					// Thêm khoảng nghỉ giữa các lần gọi API để tránh rate limiting
					select {
					case <-taskCtx.Done():
						p.Logger.Warn("Context cancelled during story parsing")
						return
					case <-time.After(5 * time.Second):
						// Tiếp tục với người dùng tiếp theo
					}
				}

				p.Logger.Info("Completed scheduled story parsing")
			},
		),
	)
	if err != nil {
		return fmt.Errorf("failed to schedule story parsing: %w", err)
	}

	// Bắt đầu scheduler và duy trì nó chạy
	scheduler.Start()

	// Theo dõi khi context bị hủy để dừng scheduler
	go func() {
		<-ctx.Done()
		p.Logger.Info("Stopping story parsing scheduler")
		if err := scheduler.Shutdown(); err != nil {
			p.Logger.Error("Failed to shut down scheduler", "error", err)
		}
	}()

	return nil
}

// ParseUserReelStories phân tích stories của một người dùng cụ thể
func (p *ParserImpl) ParseUserReelStories(ctx context.Context, username string) error {
	p.Logger.Info("Parsing reel stories", "username", username)

	// Thêm context timeout để tránh phân tích quá lâu
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Truy cập profile Instagram
	profile, err := p.Instagram.VisitProfile(username)
	if err != nil {
		return fmt.Errorf("failed to visit profile %s: %w", username, err)
	}

	// Lấy danh sách stories
	stories, err := profile.Stories()
	if err != nil {
		return fmt.Errorf("failed to get stories for %s: %w", username, err)
	}

	if stories == nil || len(stories.Reel.Items) == 0 {
		p.Logger.Info("No stories found for user", "username", username)
		return nil
	}

	p.Logger.Info("Found stories", "username", username, "count", len(stories.Reel.Items))

	// Phân tích các stories
	return p.ParseStories(stories.Reel.Items)
}

// ParseStories xử lý danh sách các stories
func (p *ParserImpl) ParseStories(stories []*goinsta.Item) error {
	if len(stories) == 0 {
		return nil
	}

	// Giới hạn số lượng goroutines cùng lúc
	semaphore := make(chan struct{}, 5)
	var wg sync.WaitGroup
	var errsMutex sync.Mutex
	var errs []error

	// Đếm số lượng stories đã xử lý
	var processed, skipped, failed int
	var statsMutex sync.Mutex

	p.Logger.Info("Parsing stories", "count", len(stories))

	for _, story := range stories {
		// Ngay lặp từng item
		wg.Add(1)
		semaphore <- struct{}{} // Lấy token từ semaphore

		go func(item *goinsta.Item) {
			defer func() {
				<-semaphore // Trả lại token vào semaphore
				wg.Done()

				// Xử lý panic nếu có
				if r := recover(); r != nil {
					errsMutex.Lock()
					errs = append(errs, fmt.Errorf("panic in story processing: %v", r))
					errsMutex.Unlock()

					statsMutex.Lock()
					failed++
					statsMutex.Unlock()
				}
			}()

			// Kiểm tra xem story đã được xử lý chưa
			exists, err := p.checkStoryExists(item)
			if err != nil {
				errsMutex.Lock()
				errs = append(errs, fmt.Errorf("failed to check story existence: %w", err))
				errsMutex.Unlock()

				statsMutex.Lock()
				failed++
				statsMutex.Unlock()
				return
			}

			if exists {
				p.Logger.Debug("Story already processed", "storyID", item.ID)
				statsMutex.Lock()
				skipped++
				statsMutex.Unlock()
				return
			}

			// Xử lý story
			if err := p.processStoryItem(item); err != nil {
				errsMutex.Lock()
				errs = append(errs, fmt.Errorf("failed to process story %s: %w", item.ID, err))
				errsMutex.Unlock()

				statsMutex.Lock()
				failed++
				statsMutex.Unlock()
				return
			}

			statsMutex.Lock()
			processed++
			statsMutex.Unlock()
		}(story)
	}

	// Đợi tất cả các goroutines hoàn thành
	wg.Wait()
	close(semaphore)

	// Tổng kết kết quả
	p.Logger.Info("Story parsing completed",
		"total", len(stories),
		"processed", processed,
		"skipped", skipped,
		"failed", failed)

	// Trả về lỗi nếu có
	if len(errs) > 0 {
		// Chỉ trả về lỗi đầu tiên với thông tin số lượng lỗi tổng cộng
		return fmt.Errorf("encountered %d errors during story parsing, first error: %w",
			len(errs), errs[0])
	}

	return nil
}

// checkStoryExists kiểm tra xem story đã được xử lý trước đó chưa
func (p *ParserImpl) checkStoryExists(item *goinsta.Item) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	storyID, ok := item.ID.(string)
	if !ok {
		return false, fmt.Errorf("invalid story ID type: expected string, got %T", item.ID)
	}

	_, err := p.StoryRepo.GetByStoryID(ctx, storyID)
	if err != nil {
		if errors.Is(err, storyRepo.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// processStoryItem xử lý một story item cụ thể
func (p *ParserImpl) processStoryItem(item *goinsta.Item) error {
	// Lưu thông tin story vào DB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert ID to string
	storyID, ok := item.ID.(string)
	if !ok {
		return fmt.Errorf("invalid story ID type: expected string, got %T", item.ID)
	}

	story := domain.Story{
		StoryID:   storyID,
		UserName:  item.User.Username,
		CreatedAt: time.Unix(item.TakenAt, 0),
	}

	if err := p.StoryRepo.Create(ctx, story); err != nil {
		// Nếu đã tồn tại thì bỏ qua, không cần xử lý lại
		if errors.Is(err, storyRepo.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("failed to save story: %w", err)
	}

	// Xử lý media trong story
	if len(item.Videos) > 0 {
		// Xử lý video
		videoURL := item.Videos[0].URL
		if videoURL != "" {
			p.Logger.Info("Processing video story",
				"username", item.User.Username,
				"url", videoURL)

			if err := p.downloadAndSendMedia(videoURL, 2); err != nil {
				return fmt.Errorf("failed to process video story: %w", err)
			}
		}
	} else if len(item.Images.Versions) > 0 {
		// Xử lý ảnh
		imageURL := item.Images.Versions[0].URL
		if imageURL != "" {
			p.Logger.Info("Processing image story",
				"username", item.User.Username,
				"url", imageURL)

			if err := p.downloadAndSendMedia(imageURL, 1); err != nil {
				return fmt.Errorf("failed to process image story: %w", err)
			}
		}
	}

	return nil
}

// downloadAndSendMedia tải xuống và gửi media đến Telegram
func (p *ParserImpl) downloadAndSendMedia(url string, mediaType int) error {
	// Tạo HTTP client với timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Gửi yêu cầu HTTP để tải xuống media
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download media: %w", err)
	}
	defer resp.Body.Close()

	// Kiểm tra status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download media, status code: %d", resp.StatusCode)
	}

	// Đọc dữ liệu media
	mediaData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read media data: %w", err)
	}

	// Kiểm tra xem có dữ liệu không
	if len(mediaData) == 0 {
		return fmt.Errorf("empty media data")
	}

	// Tạo FileBytes cho Telegram
	fileBytes := tgbotapi.FileBytes{
		Name:  "media",
		Bytes: mediaData,
	}

	// Gửi media đến Telegram
	if err := p.Telegram.SendFileToChannel(fileBytes, mediaType); err != nil {
		return fmt.Errorf("failed to send media to Telegram: %w", err)
	}

	return nil
}
