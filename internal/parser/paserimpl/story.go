package paserimpl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
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

	// Lên lịch nhiệm vụ với khoảng thời gian hợp lý và ngẫu nhiên hơn
	_, err = scheduler.NewJob(
		gocron.DurationRandomJob(
			time.Hour*3,    // Tăng khoảng thời gian tối thiểu lên 3 giờ
			time.Hour*24*2, // Tối đa 2 ngày
		),
		gocron.NewTask(
			func() {
				// Kiểm tra xem context đã bị hủy chưa
				if ctx.Err() != nil {
					p.Logger.Info("Context cancelled, stopping story parsing schedule")
					return
				}

				// Tạo context con với timeout dài hơn để thực hiện chậm hơn
				taskCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
				defer cancel()

				usernames := strings.Split(p.Config.Instagram.UsersParse, ";")
				p.Logger.Info("Starting scheduled story parsing", "userCount", len(usernames))

				// Xáo trộn danh sách usernames để không theo thứ tự cố định
				shuffledUsernames := shuffleUsernames(usernames)

				// Xử lý từng username với độ trễ ngẫu nhiên giữa các user
				for i, username := range shuffledUsernames {
					username = strings.TrimSpace(username)
					if username == "" {
						continue
					}

					// Thêm độ trễ ngẫu nhiên trước khi xử lý người dùng tiếp theo
					// Trừ người dùng đầu tiên
					if i > 0 {
						// Ngẫu nhiên từ 10-45 giây giữa các user
						delay := time.Duration(10+rand.Intn(35)) * time.Second
						p.Logger.Info("Waiting before processing next user",
							"delay", delay.String(),
							"nextUsername", username)

						select {
						case <-taskCtx.Done():
							p.Logger.Warn("Context cancelled during delay between users")
							return
						case <-time.After(delay):
							// Tiếp tục sau khi đã đợi
						}
					}

					// Thêm phần giả lập hành vi người dùng
					p.simulateHumanBehavior(taskCtx, username)

					p.Logger.Info("Parsing stories for user", "username", username)
					if err := p.ParseUserReelStories(taskCtx, username); err != nil {
						p.Logger.Error("Failed to parse stories for user",
							"username", username,
							"error", err)

						// Không gửi thông báo lỗi ngay lập tức cho tất cả, đợi một khoảng thời gian ngẫu nhiên
						time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)

						p.Telegram.SendMessageToUser(fmt.Sprintf("Failed to parse stories for %s: %s",
							username, err.Error()))
					} else {
						p.Logger.Info("Successfully parsed stories for user", "username", username)
					}

					// Thêm khoảng nghỉ có độ dài ngẫu nhiên hơn giữa các lần gọi API để tránh rate limiting
					jitter := time.Duration(rand.Intn(10)) * time.Second
					pauseDuration := 8*time.Second + jitter

					select {
					case <-taskCtx.Done():
						p.Logger.Warn("Context cancelled during story parsing")
						return
					case <-time.After(pauseDuration):
						// Tiếp tục với người dùng tiếp theo sau khi đã đợi đủ thời gian
					}
				}

				// Thêm độ trễ ngẫu nhiên sau khi hoàn thành chu kỳ
				completionJitter := time.Duration(rand.Intn(60)) * time.Second
				time.Sleep(completionJitter)

				p.Logger.Info("Completed scheduled story parsing")
			},
		),
	)
	if err != nil {
		return fmt.Errorf("failed to schedule story parsing: %w", err)
	}

	// Thêm một nhiệm vụ thứ hai vào thời điểm khác trong ngày để giả lập hành vi người dùng thực
	if err := p.addCronJob(scheduler, loc); err != nil {
		p.Logger.Warn("Failed to schedule additional parsing task", "error", err)
		// Tiếp tục dù không thể lên lịch nhiệm vụ thứ hai
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

// shuffleUsernames xáo trộn danh sách usernames để không theo thứ tự cố định
func shuffleUsernames(usernames []string) []string {
	// Tạo bản sao để không thay đổi slice gốc
	result := make([]string, len(usernames))
	copy(result, usernames)

	// Sử dụng thuật toán Fisher-Yates để xáo trộn
	rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(result) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// simulateHumanBehavior giả lập hành vi người dùng
func (p *ParserImpl) simulateHumanBehavior(ctx context.Context, username string) {
	// Tạo context con với timeout
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// Truy cập profile trước để "xem" trước khi lấy stories
	_, err := p.Instagram.VisitProfile(username)
	if err != nil {
		p.Logger.Warn("Failed to visit profile during human behavior simulation",
			"username", username,
			"error", err)
		return
	}

	// Đợi một khoảng thời gian ngẫu nhiên như người dùng đang xem profile
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(2+rand.Intn(4)) * time.Second):
		// Tiếp tục sau khi đã "xem" profile
	}

	// Có thể thực hiện một số tác vụ đơn giản để giả lập việc xem profile
	// thay vì cố gắng truy cập feed items
	time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
}

// Thêm job vào lịch trình với khoảng thời gian cố định
func (p *ParserImpl) addCronJob(scheduler gocron.Scheduler, loc *time.Location) error {
	_, err := scheduler.NewJob(
		gocron.DurationJob(
			4*time.Hour, // Chạy mỗi 4 giờ
		),
		gocron.NewTask(
			func() {
				// Chọn ngẫu nhiên một số ít người dùng để kiểm tra
				usernames := strings.Split(p.Config.Instagram.UsersParse, ";")
				if len(usernames) == 0 {
					return
				}

				// Chọn ngẫu nhiên 1-2 người dùng
				maxUsers := min(2, len(usernames))
				if maxUsers <= 0 {
					return
				}

				numUsers := 1 + rand.Intn(maxUsers)

				// Fisher-Yates shuffle
				for i := 0; i < len(usernames); i++ {
					j := rand.Intn(i + 1)
					usernames[i], usernames[j] = usernames[j], usernames[i]
				}

				selectedUsers := usernames[:numUsers]

				for _, username := range selectedUsers {
					username = strings.TrimSpace(username)
					if username == "" {
						continue
					}

					// Thêm độ trễ ngẫu nhiên
					time.Sleep(time.Duration(5+rand.Intn(15)) * time.Second)

					taskCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
					p.simulateHumanBehavior(taskCtx, username)
					p.ParseUserReelStories(taskCtx, username)
					cancel()
				}
			},
		),
	)
	return err
}

// ParseUserReelStories phân tích stories của một người dùng cụ thể
func (p *ParserImpl) ParseUserReelStories(ctx context.Context, username string) error {
	p.Logger.Info("Parsing reel stories", "username", username)

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

// SaveHighlight saves a highlight to the repository
func (p *ParserImpl) SaveHighlight(highlight domain.Highlights) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.Logger.Info("Saving highlight",
		"username", highlight.UserName,
		"mediaURL", highlight.MediaURL)

	err := p.HighlightsRepo.Create(ctx, highlight)
	if err != nil {
		return fmt.Errorf("failed to save highlight: %w", err)
	}

	return nil
}

// SaveCurrentStory saves a current story to the repository
func (p *ParserImpl) SaveCurrentStory(currentStory domain.CurrentStory) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.Logger.Info("Saving current story",
		"username", currentStory.UserName,
		"mediaURL", currentStory.MediaURL)

	err := p.CurrentStoryRepo.Create(ctx, currentStory)
	if err != nil {
		return fmt.Errorf("failed to save current story: %w", err)
	}

	return nil
}

// ClearCurrentStories removes all current stories for a username
func (p *ParserImpl) ClearCurrentStories(username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.Logger.Info("Clearing current stories for user", "username", username)

	err := p.CurrentStoryRepo.DeleteByUserName(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to clear current stories for %s: %w", username, err)
	}

	return nil
}
