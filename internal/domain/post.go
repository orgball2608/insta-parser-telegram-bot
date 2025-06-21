package domain

import "time"

type PostItem struct {
	ID        string    // Post ID from Instagram
	PostURL   string    // URL to the post
	URL       string    // Alias for PostURL for compatibility
	Username  string    // Instagram username
	Caption   string    // Post caption
	MediaURLs []string  // URLs of media (images/videos)
	IsVideo   bool      // Whether the post is a video
	TakenAt   time.Time // When the post was taken
	Timestamp time.Time // When the post was parsed
	LikeCount int       // Number of likes
	PostedAgo string    // Human-readable time since posting
}

// For backward compatibility
func (p *PostItem) GetURL() string {
	if p.URL != "" {
		return p.URL
	}
	return p.PostURL
}
