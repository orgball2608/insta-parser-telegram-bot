package domain

import "time"

type Highlights struct {
	ID        int
	UserName  string
	MediaURL  string
	CreatedAt time.Time
}

// HighlightReel contains all stories in an album
type HighlightReel struct {
	ID    string // Album ID
	Title string
	Items []StoryItem
}

// HighlightAlbumPreview contains just enough information for user selection
type HighlightAlbumPreview struct {
	ID       string
	Title    string
	CoverURL string
}
