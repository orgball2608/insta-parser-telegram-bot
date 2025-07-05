package api_adapter

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"
	"github.com/playwright-community/playwright-go"
)

func (a *APIAdapter) GetUserPosts(ctx context.Context, userName string) ([]domain.PostItem, error) {
	a.logger.Info("Fetching user posts via reliable scraper", "username", userName)

	scraperURL := fmt.Sprintf("https://instasupersave.com/en/instagram-stories/")

	page, cleanup, err := a.newScrapingPage(ctx, scraperURL)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	typeOperation := func() error {
		return page.Type("#search-form-input", userName, playwright.PageTypeOptions{Timeout: playwright.Float(10000)})
	}
	if err = retry.Do(ctx, a.logger, "TypeUsername", typeOperation, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("could not type username after retries: %w", err)
	}
	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	clickOperation := func() error {
		return page.Click("button.search-form__button")
	}
	if err = retry.Do(ctx, a.logger, "SearchButtonClick", clickOperation, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("could not click search button: %w", err)
	}

	profileSelector := ".output-profile, .error-message"
	if _, err = page.WaitForSelector(profileSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(90000)}); err != nil {
		return nil, fmt.Errorf("profile results or error message did not load in time: %w", err)
	}

	if isPrivate, _ := page.IsVisible(".error-message"); isPrivate {
		a.logger.Warn("Account is private, cannot fetch posts", "user", userName)
		return nil, instagram.ErrPrivateAccount
	}

	postsTabSelector := "//button[contains(text(),'posts')]"
	if err := page.Click(postsTabSelector); err != nil {
		if visible, listErr := page.Locator("ul.profile-media-list").IsVisible(); !visible || listErr != nil {
			return nil, fmt.Errorf("could not click 'posts' tab and no media list found: %w", err)
		}
		a.logger.Info("Could not click 'posts' tab, but media list is visible. Proceeding.", "user", userName)
	}

	mediaItemSelector := "li.profile-media-list__item"
	if _, err = page.WaitForSelector(mediaItemSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(15000)}); err != nil {
		a.logger.Warn("No posts found for user after switching to tab", "user", userName)
		return []domain.PostItem{}, nil
	}

	postLocators, err := page.Locator(mediaItemSelector).All()
	if err != nil {
		return nil, fmt.Errorf("could not get post locators: %w", err)
	}

	var posts []domain.PostItem
	idRegex := regexp.MustCompile(`_(\d+)_`)

	for i, locator := range postLocators {
		if i >= 12 {
			break
		}

		downloadLink, err := locator.Locator("a.button__download").GetAttribute("href")
		if err != nil {
			a.logger.Warn("Could not get download link for a post, skipping", "index", i)
			continue
		}

		matches := idRegex.FindStringSubmatch(downloadLink)
		var postID string
		if len(matches) > 1 {
			postID = matches[1]
		} else {
			parts := strings.Split(filepath.Base(downloadLink), "_")
			if len(parts) > 1 {
				postID = parts[1]
			} else {
				a.logger.Warn("Could not determine a unique ID for post, skipping", "index", i, "url", downloadLink)
				continue
			}
		}

		pseudoPostURL := fmt.Sprintf("https://www.instagram.com/p/%s/", postID)

		posts = append(posts, domain.PostItem{
			ID:       postID,
			PostURL:  pseudoPostURL,
			URL:      pseudoPostURL,
			Username: userName,
		})
	}

	a.logger.Info("Successfully fetched post list from scraper", "username", userName, "count", len(posts))
	return posts, nil
}

func normalizePostURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("could not parse URL: %w", err)
	}
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	return parsedURL.String(), nil
}

func (a *APIAdapter) GetUserPost(ctx context.Context, postURL string) (*domain.PostItem, error) {
	a.logger.Info("Standardizing GetUserPost to use reliable scraper", "url", postURL)

	normalizedURL, err := normalizePostURL(postURL)
	if err != nil {
		a.logger.Warn("Failed to normalize URL, using original", "original_url", postURL, "error", err)
		normalizedURL = postURL
	}

	return a.scrapeMedia(ctx, normalizedURL, "post")
}

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

	typeOperation := func() error {
		return page.Type(inputSelector, mediaURL, playwright.PageTypeOptions{Timeout: playwright.Float(10000)})
	}
	if err = retry.Do(ctx, a.logger, "TypeMediaURL", typeOperation, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("could not type %s URL after retries: %w", mediaType, err)
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
