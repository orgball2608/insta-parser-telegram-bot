package domain

import "time"

type PostItem struct {
	PostURL   string
	Username  string
	Caption   string
	MediaURLs []string
	IsVideo   bool
	TakenAt   time.Time
}
