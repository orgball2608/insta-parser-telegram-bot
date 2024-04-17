package domain

import "time"

type Story struct {
	ID        int
	StoryID   string
	UserName  string
	CreatedAt time.Time
}
