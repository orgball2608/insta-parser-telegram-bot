package domain

import "time"

const (
	SubscriptionTypeStory = "story"
	SubscriptionTypePost  = "post"
	SubscriptionTypeAll   = "all"
)

type Subscription struct {
	ID                int
	ChatID            int64
	InstagramUsername string
	SubscriptionType  string
	CreatedAt         time.Time
}

func IsValidSubscriptionType(subType string) bool {
	return subType == SubscriptionTypeStory ||
		subType == SubscriptionTypePost ||
		subType == SubscriptionTypeAll
}
