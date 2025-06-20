package api_adapter

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"
	"github.com/playwright-community/playwright-go"
)

func (a *APIAdapter) GetUserPost(ctx context.Context, postURL string) (*domain.PostItem, error) {
	return a.scrapePost(ctx, postURL)
}

func (a *APIAdapter) scrapePost(ctx context.Context, postURL string) (*domain.PostItem, error) {
	a.logger.Info("Scraping post", "url", postURL)
	page, cleanup, err := a.newScrapingPage(ctx, "https://instasupersave.com/en/instagram-story-viewer/")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err = page.Type("#search-form-input", postURL, playwright.PageTypeOptions{Timeout: playwright.Float(10000)}); err != nil {
		return nil, fmt.Errorf("could not type post URL: %w", err)
	}
	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	clickOperation := func() error {
		return page.Click("button.search-form__button")
	}
	if err = retry.Do(ctx, a.logger, "SearchPostClick", clickOperation, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("could not click search button for post: %w", err)
	}

	resultSelector := "div.output-list, .error-message"
	if _, err = page.WaitForSelector(resultSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(90000)}); err != nil {
		screenshotPath := fmt.Sprintf("tmp/error_screenshot_post_%d.png", time.Now().Unix())
		page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(screenshotPath), FullPage: playwright.Bool(true)})
		a.logger.Error("Timeout waiting for post result, screenshot saved", "path", screenshotPath, "error", err)
		return nil, fmt.Errorf("post results or error message did not load in time: %w", err)
	}

	isError, _ := page.IsVisible(".error-message")
	if isError {
		errorText, _ := page.InnerText(".error-message")
		a.logger.Warn("Error message displayed for post", "url", postURL, "message", errorText)
		return nil, fmt.Errorf("failed to get post: %s", errorText)
	}

	postItem := &domain.PostItem{PostURL: postURL}

	caption, err := page.InnerText("div.output-list__caption p")
	if err == nil {
		postItem.Caption = caption
	} else {
		a.logger.Warn("Could not find post caption", "url", postURL, "error", err)
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
	postItem.MediaURLs = mediaURLs

	for _, url := range mediaURLs {
		if strings.Contains(url, ".mp4") {
			postItem.IsVideo = true
			break
		}
	}

	a.logger.Info("Successfully scraped post", "url", postURL, "media_count", len(postItem.MediaURLs))
	return postItem, nil
}
