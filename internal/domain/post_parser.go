package domain

import "time"

type PostParser struct {
	ID        int
	PostID    string
	Username  string
	PostURL   string
	CreatedAt time.Time
}
