package domain

import "time"

type Highlights struct {
	ID        int
	UserName  string
	MediaURL  string
	CreatedAt time.Time
}

type HighlightReel struct {
	Title string
	Items []StoryItem
}
