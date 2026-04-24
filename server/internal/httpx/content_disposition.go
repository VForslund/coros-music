package httpx

import (
	"fmt"
	"net/url"
	"strings"
)

// FormatAttachmentContentDisposition returns an ASCII-safe attachment header value
// with an RFC 5987 UTF-8 filename for clients that support it.
func FormatAttachmentContentDisposition(filename string) string {
	if strings.TrimSpace(filename) == "" {
		filename = "download"
	}

	return fmt.Sprintf(`attachment; filename=%q; filename*=UTF-8''%s`, asciiFilenameFallback(filename), url.PathEscape(filename))
}

func asciiFilenameFallback(filename string) string {
	var b strings.Builder
	for _, r := range filename {
		switch {
		case r >= 0x20 && r <= 0x7e && r != '"' && r != '\\':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	out := strings.TrimSpace(b.String())
	if out == "" {
		return "download"
	}

	return out
}
