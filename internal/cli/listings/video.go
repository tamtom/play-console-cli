package listings

import (
	"fmt"
	"regexp"
)

var youtubeRegexp = regexp.MustCompile(`^(https?://)?(www\.)?(youtube\.com/watch\?v=|youtu\.be/)[\w-]+`)

// ValidateVideoURL validates a YouTube URL. Empty string is allowed (clears video).
func ValidateVideoURL(url string) error {
	if url == "" {
		return nil
	}
	if !youtubeRegexp.MatchString(url) {
		return fmt.Errorf("invalid YouTube URL: %s (expected youtube.com/watch?v=... or youtu.be/...)", url)
	}
	return nil
}
