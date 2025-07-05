package domain

import "time"

type PostItem struct {
	ID        string
	PostURL   string
	URL       string
	Username  string
	Caption   string
	MediaURLs []string
	IsVideo   bool
	TakenAt   time.Time
	Timestamp time.Time
	LikeCount int
	PostedAgo string
}

func (p *PostItem) GetURL() string {
	if p.URL != "" {
		return p.URL
	}
	return p.PostURL
}
