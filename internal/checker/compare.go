package checker

import (
	"project/mangachapter/internal/source"
)

// HasNewChapter determines if the fetched chapter is newer than the stored one.
// Returns true if there's a new chapter to notify about.
func HasNewChapter(storedNumValue float64, fetched *source.ChapterInfo) bool {
	if fetched == nil {
		return false
	}
	// If no baseline chapter stored (0), treat as new
	if storedNumValue == 0 {
		return true
	}
	return fetched.NumValue > storedNumValue
}