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
	return a.scrapeMedia(ctx, postURL, "post")
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

	caption, err := page.InnerText(".output-list__caption p, .output-list__info-text")
	if err == nil {
		mediaItem.Caption = caption
	} else {
		a.logger.Warn("Could not find media caption", "url", mediaURL, "error", err)
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

	a.logger.Info("Successfully scraped media", "type", mediaType, "url", mediaURL, "media_count", len(mediaItem.MediaURLs))
	return mediaItem, nil
}
