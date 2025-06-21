package domain

import "time"

// Subscription types
const (
	SubscriptionTypeStory = "story"
	SubscriptionTypePost  = "post"
	SubscriptionTypeAll   = "all"
)

type Subscription struct {
	ID                int
	ChatID            int64
	InstagramUsername string
	SubscriptionType  string // Added field for subscription type
	CreatedAt         time.Time
}

// IsValidSubscriptionType checks if the provided subscription type is valid
func IsValidSubscriptionType(subType string) bool {
	return subType == SubscriptionTypeStory ||
		subType == SubscriptionTypePost ||
		subType == SubscriptionTypeAll
}
