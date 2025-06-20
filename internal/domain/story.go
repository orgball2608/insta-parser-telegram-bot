package domain

import "time"

type Story struct {
	ID        int
	StoryID   string
	UserName  string
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
