package api_adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type apiStoryResponse struct {
	Stories []struct {
		ID       string `json:"id"`
		MediaURL string `json:"media_url"`
		Type     string `json:"type"`
		TakenAt  int64  `json:"taken_at"`
	} `json:"stories"`
}

type apiHighlightResponse struct {
	Highlights []struct {
		Title string `json:"title"`
		Items []struct {
			ID       string `json:"id"`
			MediaURL string `json:"media_url"`
			Type     string `json:"type"`
			TakenAt  int64  `json:"taken_at"`
		} `json:"items"`
	} `json:"highlights"`
}

type Opts struct {
	fx.In
	Config *config.Config
	Logger logger.Logger
}

type APIAdapter struct {
	client *http.Client
	config *config.Config
	logger logger.Logger
}

func New(opts Opts) instagram.Client {
	return &APIAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
		config: opts.Config,
		logger: opts.Logger,
	}
}

func (a *APIAdapter) GetUserStories(userName string) ([]domain.StoryItem, error) {
	url := fmt.Sprintf("%s/users/%s/stories", a.config.ThirdPartyAPI.BaseURL, userName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.config.ThirdPartyAPI.APIKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stories from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp apiStoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	var storyItems []domain.StoryItem
	for _, s := range apiResp.Stories {
		storyItems = append(storyItems, domain.StoryItem{
			ID:        s.ID,
			MediaURL:  s.MediaURL,
			MediaType: domain.MediaType(s.Type),
			TakenAt:   time.Unix(s.TakenAt, 0),
			Username:  userName,
		})
	}

	return storyItems, nil
}

func (a *APIAdapter) GetUserHighlights(userName string) ([]domain.HighlightReel, error) {
	url := fmt.Sprintf("%s/users/%s/highlights", a.config.ThirdPartyAPI.BaseURL, userName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.config.ThirdPartyAPI.APIKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch highlights from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp apiHighlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	var highlightReels []domain.HighlightReel
	for _, h := range apiResp.Highlights {
		var items []domain.StoryItem
		for _, item := range h.Items {
			items = append(items, domain.StoryItem{
				ID:        item.ID,
				MediaURL:  item.MediaURL,
				MediaType: domain.MediaType(item.Type),
				TakenAt:   time.Unix(item.TakenAt, 0),
				Username:  userName,
			})
		}
		highlightReels = append(highlightReels, domain.HighlightReel{
			Title: h.Title,
			Items: items,
		})
	}
	return highlightReels, nil
}

var _ instagram.Client = (*APIAdapter)(nil)
