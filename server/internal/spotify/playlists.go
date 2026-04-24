package spotify

import (
	"strings"
	"unicode"
)

// sanitizeFilename builds "{TrackID} — {Artist} - {Title}.mp3" with filesystem-safe chars.
func sanitizeFilename(id, artist, title string) string {
	clean := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '(' || r == ')' || r == '\'' {
				b.WriteRune(r)
			}
		}
		return strings.TrimSpace(b.String())
	}
	return id + " — " + clean(artist) + " - " + clean(title) + ".mp3"
}

