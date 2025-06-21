package domain

import "time"

// PostParser represents a parsed Instagram post
type PostParser struct {
	ID        int
	PostID    string
	Username  string
	PostURL   string
	CreatedAt time.Time
}
