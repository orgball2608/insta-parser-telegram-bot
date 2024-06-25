package instagramimpl

import (
	"github.com/Davincible/goinsta/v3"
)

func (ig *IgImpl) GetUserStories(userName string) ([]*goinsta.Item, error) {
	ig.Logger.Info("Get stories for username", "userName", userName)
	profile, err := ig.Client.VisitProfile(userName)
	if err != nil {
		ig.Logger.Error("Visit profile error", "Error", err)
		return nil, err
	}

	stories := profile.Stories.Reel

	return stories.Items, nil
}

func (ig *IgImpl) GetUserHighlights(userName string) ([]*goinsta.Reel, error) {
	ig.Logger.Info("Get stories", "userName", userName)
	profile, err := ig.Client.VisitProfile(userName)
	if err != nil {
		ig.Logger.Error("Visit profile error", "Error", err)
		return nil, err
	}

	stories := profile.Highlights
	return stories, nil
}
