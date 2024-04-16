package domain

import "time"

type Story struct {
	ID        int
	StoryID   string
	UserName  string
	Result    bool
	CreatedAt time.Time
}
