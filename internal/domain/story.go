package domain

import "time"

type Story struct {
	ID        int
	StoryID   string
	UserName  string
	CreatedAt time.Time
}

type Highlights struct {
	ID        int
	UserName  string
	MediaURL  string
	CreatedAt time.Time
}

type CurrentStory struct {
	ID        int
	UserName  string
	MediaURL  string
	CreatedAt time.Time
}

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

type StoryItem struct {
	ID        string
	MediaURL  string
	MediaType MediaType
	TakenAt   time.Time
	Username  string
}

type HighlightReel struct {
	Title string
	Items []StoryItem
}

type PostItem struct {
	PostURL   string
	Username  string
	Caption   string
	MediaURLs []string
	IsVideo   bool
	TakenAt   time.Time
}
