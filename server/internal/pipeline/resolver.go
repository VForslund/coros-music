package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Resolve uses yt-dlp to find the best audio URL for a search query.
// Returns the direct audio URL that can be piped into FFmpeg.
func Resolve(ctx context.Context, query string) (string, error) {
	ytdlpPath := os.Getenv("COROS_YTDLP_PATH")
	if ytdlpPath == "" {
		ytdlpPath = "yt-dlp"
	}

	cmd := exec.CommandContext(ctx, ytdlpPath,
		"--no-playlist",
		"--print", "urls",
		"-f", "bestaudio",
		fmt.Sprintf("ytsearch1:%s", query),
	)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp resolve failed for %q: %w", query, err)
	}

	url := strings.TrimSpace(string(out))
	if url == "" {
		return "", fmt.Errorf("yt-dlp returned empty URL for %q", query)
	}

	return url, nil
}

// CheckBinary returns true if the given binary is available on PATH.
func CheckBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

