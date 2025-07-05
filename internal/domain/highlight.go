package domain

import "time"

type Highlights struct {
	ID        int
	UserName  string
	MediaURL  string
	CreatedAt time.Time
}

type HighlightReel struct {
	ID    string
	Title string
	Items []StoryItem
}

type HighlightAlbumPreview struct {
	ID       string
	Title    string
	CoverURL string
}
