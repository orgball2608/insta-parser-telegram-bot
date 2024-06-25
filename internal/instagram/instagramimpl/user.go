package instagramimpl

import (
	"github.com/Davincible/goinsta/v3"
)

func (ig *IgImpl) Login() error {
	if err := ig.ReloadSession(); err != nil {
		err := ig.Client.Login()
		if err != nil {
			ig.Logger.Error("Login error", "Error", err)
			return err
		}

		ig.Logger.Info("Successfully logged in by username and password")

		defer func(client *goinsta.Instagram, path string) {
			err := client.Export(path)
			if err != nil {
				ig.Logger.Error("Couldn't save the session", "Error", err)
			}
		}(ig.Client, ig.Config.Instagram.SessionPath)
	}

	return nil
}

func (ig *IgImpl) ReloadSession() error {
	instagram, err := goinsta.Import(ig.Config.Instagram.SessionPath)
	if err != nil {
		ig.Logger.Error("Couldn't recover the session", "Error", err)
		return err
	}

	ig.Client = instagram

	ig.Logger.Info("Successfully logged in by session")
	return nil
}
