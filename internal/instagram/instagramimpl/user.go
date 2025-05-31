package instagramimpl

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Davincible/goinsta/v3"
)

// Login attempts to connect to Instagram, first trying to load from an existing session,
// or logging in with credentials if the session isn't available.
func (ig *IgImpl) Login() error {
	// First try loading from session
	if err := ig.ReloadSession(); err == nil {
		// Verify the session is valid by making a simple API call
		if ig.validateSession() {
			ig.Logger.Info("Successfully logged in using existing session")
			return nil
		}
		ig.Logger.Warn("Session loaded but appears to be invalid, attempting fresh login")
	}

	// Fall back to username/password login
	ig.Logger.Info("Attempting to log in with credentials")

	// Create a fresh Instagram client with credentials
	ig.Client = goinsta.New(ig.Config.Instagram.Username, ig.Config.Instagram.Password)

	// Add retry logic for login
	var loginErr error
	for attempt := 1; attempt <= 3; attempt++ {
		loginErr = ig.Client.Login()
		if loginErr == nil {
			break
		}

		ig.Logger.Error("Login attempt failed",
			"attempt", attempt,
			"error", loginErr)

		// Only sleep if we're going to try again
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if loginErr != nil {
		return fmt.Errorf("failed to log in after multiple attempts: %w", loginErr)
	}

	ig.Logger.Info("Successfully logged in with credentials")

	// Save the session for future use
	if err := ig.saveSession(); err != nil {
		ig.Logger.Warn("Failed to save Instagram session", "error", err)
		// Continue despite session save failure
	}

	return nil
}

// ReloadSession attempts to load an existing Instagram session
func (ig *IgImpl) ReloadSession() error {
	// Check if a session file exists first
	if _, err := os.Stat(ig.Config.Instagram.SessionPath); os.IsNotExist(err) {
		return fmt.Errorf("session file not found: %w", err)
	}

	instagram, err := goinsta.Import(ig.Config.Instagram.SessionPath)
	if err != nil {
		return fmt.Errorf("failed to import session: %w", err)
	}

	// Update the client instance
	ig.Client = instagram
	return nil
}

// validateSession checks if the current Instagram session is valid
func (ig *IgImpl) validateSession() bool {
	if ig.Client == nil {
		return false
	}

	// Try to get account info to validate the session
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a channel for timeout handling
	done := make(chan bool, 1)
	var valid bool

	go func() {
		defer func() {
			// Recover from any panics in the Instagram client
			if r := recover(); r != nil {
				ig.Logger.Error("Panic in Instagram session validation", "panic", r)
				valid = false
			}
			done <- true
		}()

		// Try to access account info
		err := ig.Client.Account.Sync()
		valid = err == nil
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		return valid
	case <-ctx.Done():
		ig.Logger.Warn("Session validation timed out")
		return false
	}
}

// saveSession exports the current Instagram session to a file
func (ig *IgImpl) saveSession() error {
	if ig.Client == nil {
		return fmt.Errorf("no active Instagram session to save")
	}

	// Ensure directory exists
	dir := ig.Config.Instagram.SessionPath
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create session directory: %w", err)
		}
	}

	// Export the session
	if err := ig.Client.Export(ig.Config.Instagram.SessionPath); err != nil {
		return fmt.Errorf("failed to export session: %w", err)
	}

	ig.Logger.Info("Instagram session saved successfully",
		"path", ig.Config.Instagram.SessionPath)
	return nil
}

func (ig *IgImpl) VisitProfile(username string) (*goinsta.User, error) {
	user, err := ig.Client.Profiles.ByName(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile %s: %w", username, err)
	}
	return user, nil
}
