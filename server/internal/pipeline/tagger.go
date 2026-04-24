package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// TagMP3 writes ID3v2 tags using FFmpeg (avoids iconv dependency from id3-go).
// Returns the tagged MP3 bytes.
func TagMP3(mp3Data []byte, title, artist, album string) ([]byte, error) {
	ffmpegPath := os.Getenv("COROS_FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	// FFmpeg can write ID3 tags via metadata flags, reading from stdin and writing to stdout
	cmd := exec.Command(ffmpegPath,
		"-i", "pipe:0",
		"-codec", "copy",
		"-metadata", fmt.Sprintf("title=%s", title),
		"-metadata", fmt.Sprintf("artist=%s", artist),
		"-metadata", fmt.Sprintf("album=%s", album),
		"-f", "mp3",
		"-loglevel", "error",
		"pipe:1",
	)

	cmd.Stdin = bytes.NewReader(mp3Data)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg tag: %w", err)
	}

	return out.Bytes(), nil
}
