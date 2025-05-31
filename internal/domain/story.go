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
