package api_adapter

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"github.com/playwright-community/playwright-go"
	"go.uber.org/fx"
)

type Opts struct {
	fx.In
	Config *config.Config
	Logger logger.Logger
}

type APIAdapter struct {
	config *config.Config
	logger logger.Logger
}

func New(opts Opts) instagram.Client {
	return &APIAdapter{
		config: opts.Config,
		logger: opts.Logger,
	}
}

func (a *APIAdapter) GetUserStories(userName string) ([]domain.StoryItem, error) {
	links, err := a.scrapeStoryLinks(userName)
	if err != nil {
		return nil, err
	}
	var storyItems []domain.StoryItem
	for _, link := range links {
		storyItems = append(storyItems, domain.StoryItem{
			MediaURL: link,
			Username: userName,
		})
	}
	return storyItems, nil
}

func (a *APIAdapter) GetUserHighlights(userName string, processorFunc instagram.HighlightReelProcessorFunc) error {
	return a.scrapeHighlightLinks(userName, processorFunc)
}

func (a *APIAdapter) scrapeStoryLinks(userName string) ([]string, error) {
	a.logger.Info("Starting playwright to scrape stories", "user", userName)

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-setuid-sandbox"},
	})
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return nil, fmt.Errorf("could not create browser context: %w", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %w", err)
	}

	if _, err = page.Goto("https://instasupersave.com/en/instagram-stories/", playwright.PageGotoOptions{Timeout: playwright.Float(60000)}); err != nil {
		return nil, fmt.Errorf("could not goto page: %w", err)
	}

	if err = page.Type("#search-form-input", userName, playwright.PageTypeOptions{Timeout: playwright.Float(10000)}); err != nil {
		return nil, fmt.Errorf("could not type username: %w", err)
	}

	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	if err = page.Click("button.search-form__button"); err != nil {
		return nil, fmt.Errorf("could not click search button: %w", err)
	}

	combinedSelector := ".output-profile, .error-message"
	if _, err = page.WaitForSelector(combinedSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(45000)}); err != nil {
		return nil, fmt.Errorf("search results or error message did not load in time: %w", err)
	}

	isPrivate, _ := page.IsVisible(".error-message")
	if isPrivate {
		a.logger.Warn("Account is private, cannot scrape stories", "user", userName)
		return nil, instagram.ErrPrivateAccount
	}

	a.logger.Info("Processing 'stories' tab...")
	tabSelector := "//button[contains(text(),'stories')]"
	if err := page.Click(tabSelector); err != nil {
		return nil, fmt.Errorf("could not click stories tab: %w", err)
	}
	time.Sleep(2 * time.Second)
	storyLinks, err := scrollAndExtractAllLinks(page)
	if err != nil {
		a.logger.Error("Failed to extract from stories tab", "error", err)
		return []string{}, nil
	}
	return storyLinks, nil
}

func (a *APIAdapter) scrapeHighlightLinks(userName string, processorFunc instagram.HighlightReelProcessorFunc) error {
	a.logger.Info("Starting playwright to scrape highlights", "user", userName)

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-setuid-sandbox"},
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return fmt.Errorf("could not create browser context: %w", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}

	if _, err = page.Goto("https://instasupersave.com/en/instagram-stories/", playwright.PageGotoOptions{Timeout: playwright.Float(60000)}); err != nil {
		return fmt.Errorf("could not goto page: %w", err)
	}

	if err = page.Type("#search-form-input", userName, playwright.PageTypeOptions{Timeout: playwright.Float(10000)}); err != nil {
		return fmt.Errorf("could not type username: %w", err)
	}

	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	if err = page.Click("button.search-form__button"); err != nil {
		return fmt.Errorf("could not click search button: %w", err)
	}

	combinedSelector := ".output-profile, .error-message"
	if _, err = page.WaitForSelector(combinedSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(45000)}); err != nil {
		return fmt.Errorf("search results or error message did not load in time: %w", err)
	}

	isPrivate, _ := page.IsVisible(".error-message")
	if isPrivate {
		a.logger.Warn("Account is private, cannot scrape highlights", "user", userName)
		return instagram.ErrPrivateAccount
	}

	a.logger.Info("Processing 'highlights' tab...")
	tabSelector := "//button[contains(text(),'highlights')]"
	if err := page.Click(tabSelector); err != nil {
		return fmt.Errorf("could not click highlights tab: %w", err)
	}

	highlightAlbumSelector := "button.highlight__button"
	if _, err = page.WaitForSelector(highlightAlbumSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(15000)}); err != nil {
		a.logger.Warn("Highlight albums did not appear.", "error", err)
		return nil
	}

	albumCount, err := page.Locator(highlightAlbumSelector).Count()
	if err != nil {
		return fmt.Errorf("could not count highlight albums: %w", err)
	}
	a.logger.Info("Found highlight albums.", "count", albumCount)

	for i := 0; i < albumCount; i++ {
		currentAlbum := page.Locator(highlightAlbumSelector).Nth(i)

		albumTitle, _ := currentAlbum.Locator("p.highlight__title").InnerText()
		a.logger.Info("Processing album", "index", i+1, "title", albumTitle)

		if err := currentAlbum.Click(playwright.LocatorClickOptions{Timeout: playwright.Float(5000)}); err != nil {
			a.logger.Warn("Could not click on album, skipping.", "title", albumTitle, "error", err)
			continue
		}

		albumLinks, err := scrollAndExtractAllLinks(page)
		if err != nil {
			a.logger.Error("Failed to extract links for album", "title", albumTitle, "error", err)
			continue
		}

		var highlightItems []domain.StoryItem
		for _, link := range albumLinks {
			highlightItems = append(highlightItems, domain.StoryItem{
				MediaURL: link,
				Username: userName,
			})
		}

		reel := domain.HighlightReel{
			Title: albumTitle,
			Items: highlightItems,
		}

		if err := processorFunc(reel); err != nil {
			a.logger.Error("Processor function returned an error, stopping highlight processing", "error", err)
			return err
		}

		a.logger.Info("Finished processing album", "title", albumTitle, "new_links", len(albumLinks))
	}

	return nil
}

func scrollAndExtractAllLinks(page playwright.Page) ([]string, error) {
	linksSet := make(map[string]bool)
	previousLinkCount := -1

	for i := 0; i < 30; i++ {
		mediaListSelector := "ul.profile-media-list"
		if _, err := page.WaitForSelector(mediaListSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(10000)}); err != nil {
			if i == 0 {
				log.Println("Media list container not found on first attempt, maybe no media.")
			}
			break
		}

		downloadButtonSelector := "a.button__download"
		locators, err := page.Locator(downloadButtonSelector).All()
		if err != nil {
			log.Printf("could not get download button locators: %v", err)
			continue
		}

		for _, locator := range locators {
			href, err := locator.GetAttribute("href")
			if err == nil && href != "" {
				linksSet[href] = true
			}
		}

		currentLinkCount := len(linksSet)
		if currentLinkCount == previousLinkCount {
			log.Printf("Scroll finished: No new links found. Total: %d", currentLinkCount)
			break
		}

		log.Printf("Scroll attempt %d: Found %d unique links (previously %d)", i+1, currentLinkCount, previousLinkCount)
		previousLinkCount = currentLinkCount

		page.Evaluate("window.scrollTo(0, document.body.scrollHeight)")

		time.Sleep(time.Duration(1500+rand.Intn(1000)) * time.Millisecond)
	}

	finalLinks := make([]string, 0, len(linksSet))
	for link := range linksSet {
		finalLinks = append(finalLinks, link)
	}

	return finalLinks, nil
}
