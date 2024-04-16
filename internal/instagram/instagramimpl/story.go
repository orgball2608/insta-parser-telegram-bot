package instagramimpl

import (
	"github.com/Davincible/goinsta/v3"
)

func (i *InstaImpl) GetUserStories(userName string) ([]*goinsta.Item, error) {
	i.logger.Info("Get stories for username: "+userName, "Username")
	profile, err := i.client.VisitProfile(userName)
	if err != nil {
		i.logger.Error("Visit profile error", "Error", err)
		return nil, err
	}

	stories := profile.Stories.Reel

	return stories.Items, nil
}
