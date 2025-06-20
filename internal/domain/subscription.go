package domain

import "time"

type Subscription struct {
	ID                int
	ChatID            int64
	InstagramUsername string
	CreatedAt         time.Time
}
