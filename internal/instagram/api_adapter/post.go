package api_adapter

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"
	"github.com/playwright-community/playwright-go"
)

// GetUserPosts retrieves the latest posts for a user
func (a *APIAdapter) GetUserPosts(ctx context.Context, userName string) ([]domain.PostItem, error) {
	a.logger.Info("Fetching posts for user", "username", userName)

	url := fmt.Sprintf("https://www.instagram.com/%s/", userName)
	page, cleanup, err := a.newScrapingPage(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer cleanup()

	// Check if account is private
	privateAccountSelector := "//h2[contains(text(), 'This Account is Private')]"
	isPrivate, err := page.IsVisible(privateAccountSelector)
	if err == nil && isPrivate {
		a.logger.Warn("Account is private, cannot fetch posts", "user", userName)
		return nil, instagram.ErrPrivateAccount
	}

	// Wait for posts to load
	postsSelector := "article a[href*='/p/']"
	if _, err = page.WaitForSelector(postsSelector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(30000),
		State:   playwright.WaitForSelectorStateAttached,
	}); err != nil {
		a.logger.Warn("No posts found or could not load posts", "user", userName, "error", err)
		return []domain.PostItem{}, nil
	}

	// Scroll to load more posts (optional, can be adjusted based on needs)
	err = scrollPageToLoadMore(page, 3) // Scroll 3 times to load more posts
	if err != nil {
		a.logger.Warn("Error while scrolling to load more posts", "error", err)
		// Continue with what we have
	}

	// Extract post URLs
	postLinks, err := page.Locator(postsSelector).All()
	if err != nil {
		return nil, fmt.Errorf("failed to locate post links: %w", err)
	}

	var posts []domain.PostItem
	for i, link := range postLinks {
		if i >= 12 { // Limit to 12 most recent posts
			break
		}

		href, err := link.GetAttribute("href")
		if err != nil {
			a.logger.Warn("Failed to get post URL", "index", i, "error", err)
			continue
		}

		postURL := "https://www.instagram.com" + href
		postID := extractPostIDFromURL(postURL)

		// Create a basic post item (we'll fetch details when needed)
		posts = append(posts, domain.PostItem{
			ID:       postID,
			URL:      postURL,
			Username: userName,
		})
	}

	a.logger.Info("Successfully fetched posts", "username", userName, "count", len(posts))
	return posts, nil
}

// GetUserPost retrieves details for a specific post
func (a *APIAdapter) GetUserPost(ctx context.Context, postURL string) (*domain.PostItem, error) {
	a.logger.Info("Fetching post details", "url", postURL)

	// Normalize URL if needed
	if !strings.Contains(postURL, "instagram.com") {
		postURL = "https://www.instagram.com" + postURL
	}

	page, cleanup, err := a.newScrapingPage(ctx, postURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer cleanup()

	// Wait for post to load
	postSelector := "article"
	if _, err = page.WaitForSelector(postSelector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(30000),
		State:   playwright.WaitForSelectorStateAttached,
	}); err != nil {
		return nil, fmt.Errorf("post not found or could not load: %w", err)
	}

	// Extract post details
	postID := extractPostIDFromURL(postURL)

	// Get username
	usernameSelector := "header a"
	username, err := page.Locator(usernameSelector).First().InnerText()
	if err != nil {
		a.logger.Warn("Failed to get username", "error", err)
		username = ""
	}
	username = strings.TrimSpace(username)

	// Get caption
	captionSelector := "ul li span"
	caption, err := page.Locator(captionSelector).First().InnerText()
	if err != nil {
		a.logger.Debug("No caption found or error", "error", err)
		caption = ""
	}

	// Get media URLs
	var mediaURLs []string

	// Try to find images
	imgSelector := "article img[src]"
	imgElements, err := page.Locator(imgSelector).All()
	if err == nil {
		for _, img := range imgElements {
			src, err := img.GetAttribute("src")
			if err == nil && src != "" && !strings.Contains(src, "profile_pic") {
				mediaURLs = append(mediaURLs, src)
			}
		}
	}

	// Try to find videos
	videoSelector := "article video source[src]"
	videoElements, err := page.Locator(videoSelector).All()
	if err == nil {
		for _, video := range videoElements {
			src, err := video.GetAttribute("src")
			if err == nil && src != "" {
				mediaURLs = append(mediaURLs, src)
			}
		}
	}

	post := &domain.PostItem{
		ID:        postID,
		URL:       postURL,
		Username:  username,
		Caption:   caption,
		MediaURLs: mediaURLs,
		Timestamp: time.Now(),
	}

	return post, nil
}

// Helper function to scroll the page to load more content
func scrollPageToLoadMore(page playwright.Page, scrollCount int) error {
	for i := 0; i < scrollCount; i++ {
		_, err := page.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`)
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

// Helper function to extract post ID from URL
func extractPostIDFromURL(url string) string {
	parts := strings.Split(url, "/p/")
	if len(parts) < 2 {
		return ""
	}

	idPart := parts[1]
	idPart = strings.Split(idPart, "/")[0]
	return idPart
}

func (a *APIAdapter) scrapeMedia(ctx context.Context, mediaURL string, mediaType string) (*domain.PostItem, error) {
	a.logger.Info("Scraping media", "type", mediaType, "url", mediaURL)

	scraperURL := "https://instasupersave.com/en/instagram-video/"

	page, cleanup, err := a.newScrapingPage(ctx, scraperURL)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cookieButtonSelector := "button.button.cookie-policy__button"
	if isVisible, _ := page.IsVisible(cookieButtonSelector); isVisible {
		a.logger.Info("Cookie policy button found, clicking it.")
		if err := page.Click(cookieButtonSelector, playwright.PageClickOptions{Timeout: playwright.Float(5000)}); err != nil {
			a.logger.Warn("Could not click cookie policy button, proceeding anyway.", "error", err)
		}
	}

	inputSelector := "#search-form-input"
	submitButtonSelector := "button.search-form__button"
	timeout := float64(30000)

	if _, err = page.WaitForSelector(inputSelector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(timeout),
	}); err != nil {
		return nil, fmt.Errorf("input field '%s' not visible: %w", inputSelector, err)
	}

	if err = page.Type(inputSelector, mediaURL, playwright.PageTypeOptions{Timeout: playwright.Float(10000)}); err != nil {
		return nil, fmt.Errorf("could not type %s URL: %w", mediaType, err)
	}

	time.Sleep(time.Duration(500+rand.Intn(500)) * time.Millisecond)

	clickOperation := func() error {
		return page.Click(submitButtonSelector)
	}
	if err = retry.Do(ctx, a.logger, "SearchMediaClick", clickOperation, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("could not click search button for %s: %w", mediaType, err)
	}

	resultSelector := "div.output-list, .output-component, .error-message"
	if _, err = page.WaitForSelector(resultSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(90000)}); err != nil {
		screenshotPath := fmt.Sprintf("tmp/error_screenshot_%s_%d.png", mediaType, time.Now().Unix())
		page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(screenshotPath), FullPage: playwright.Bool(true)})
		a.logger.Error("Timeout waiting for media result, screenshot saved", "path", screenshotPath, "error", err)
		return nil, fmt.Errorf("%s results or error message did not load in time: %w", mediaType, err)
	}

	if isError, _ := page.IsVisible(".error-message"); isError {
		errorText, _ := page.InnerText(".error-message")
		a.logger.Warn("Error message displayed for media", "url", mediaURL, "message", errorText)
		return nil, fmt.Errorf("failed to get %s: %s", mediaType, errorText)
	}

	mediaItem := &domain.PostItem{PostURL: mediaURL}

	if caption, err := page.InnerText(".output-list__caption p"); err == nil {
		mediaItem.Caption = caption
	} else {
		a.logger.Warn("Could not find media caption", "url", mediaURL, "error", err)
	}

	if avatarHref, err := page.GetAttribute(".output-list__user-avatar", "href"); err == nil {
		if u, err := url.Parse(avatarHref); err == nil {
			mediaItem.Username = strings.Trim(u.Path, "/")
		}
	}

	if likesText, err := page.InnerText(".output-list__info-like"); err == nil {
		parts := strings.Fields(likesText)
		if len(parts) > 0 {
			if likeCount, err := strconv.Atoi(strings.ReplaceAll(parts[0], ",", "")); err == nil {
				mediaItem.LikeCount = likeCount
			}
		}
	}

	if postedAgoText, err := page.InnerText(".output-list__info-time"); err == nil {
		mediaItem.PostedAgo = strings.TrimSpace(postedAgoText)
	}

	downloadLocators, err := page.Locator("a.button__download").All()
	if err != nil {
		return nil, fmt.Errorf("could not find download buttons: %w", err)
	}

	if len(downloadLocators) == 0 {
		return nil, fmt.Errorf("no download links found on the page")
	}

	var mediaURLs []string
	for _, locator := range downloadLocators {
		href, err := locator.GetAttribute("href")
		if err == nil && href != "" {
			mediaURLs = append(mediaURLs, href)
		}
	}
	mediaItem.MediaURLs = mediaURLs

	if mediaType == "reel" {
		mediaItem.IsVideo = true
	} else {
		for _, url := range mediaURLs {
			if strings.Contains(url, ".mp4") {
				mediaItem.IsVideo = true
				break
			}
		}
	}

	a.logger.Info("Successfully scraped media", "type", mediaType, "url", mediaURL, "media_count", len(mediaItem.MediaURLs), "likes", mediaItem.LikeCount, "posted_ago", mediaItem.PostedAgo)

	return mediaItem, nil
}
