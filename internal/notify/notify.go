package notify

import (
	"fmt"
	"github.com/gen2brain/beeep"
)

// Notify sends a desktop notification
func Notify(title, message string) error {
	return beeep.Notify(title, message, "")
}

// NotifyError sends an error notification
func NotifyError(repoName, errorMsg string) error {
	title := fmt.Sprintf("Autogit Paused: Error in %s", repoName)
	message := fmt.Sprintf("Merge Conflict or Network Error: %s", errorMsg)
	return Notify(title, message)
}

// NotifySuccess sends a success notification
func NotifySuccess(repoName, commitMsg string) error {
	title := fmt.Sprintf("Autogit: Committed to %s", repoName)
	message := fmt.Sprintf("Commit: %s", commitMsg)
	return Notify(title, message)
}

