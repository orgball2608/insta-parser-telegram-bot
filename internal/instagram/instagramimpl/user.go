package instagramimpl

import (
	"github.com/Davincible/goinsta/v3"
)

func (i *InstaImpl) Login() error {
	if err := i.ReloadSession(); err != nil {
		err := i.client.Login()
		if err != nil {
			i.logger.Error("Login error", "Error", err)
			return err
		}

		i.logger.Info("Successfully logged in by username and password")

		if err := i.client.Export("./goinsta-session"); err != nil {
			i.logger.Error("Couldn't save the session", "Error", err)
		}
	}

	i.logger.Info("Login by session success")
	return nil
}

func (i *InstaImpl) ReloadSession() error {
	instagram, err := goinsta.Import("./goinsta-session")
	if err != nil {
		i.logger.Error("Couldn't recover the session", "Error", err)
		return err
	}

	i.client = instagram

	i.logger.Info("Successfully logged in by session")
	return nil
}
