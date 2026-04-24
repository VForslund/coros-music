package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Transcode pipes yt-dlp audio through FFmpeg to produce a 320 kbps MP3 stream.
// The output is written directly to w (typically an io.Pipe writer or HTTP response).
func Transcode(ctx context.Context, searchQuery string, w io.Writer) error {
	ytdlpPath := os.Getenv("COROS_YTDLP_PATH")
	if ytdlpPath == "" {
		ytdlpPath = "yt-dlp"
	}
	ffmpegPath := os.Getenv("COROS_FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	// 1. yt-dlp streams best audio to stdout
	ytdlp := exec.CommandContext(ctx, ytdlpPath,
		"--no-playlist",
		"-f", "bestaudio",
		"-o", "-",
		fmt.Sprintf("ytsearch1:%s", searchQuery),
	)

	// 2. FFmpeg converts to consistent 320 kbps MP3
	ffmpeg := exec.CommandContext(ctx, ffmpegPath,
		"-i", "pipe:0",
		"-vn",
		"-codec:a", "libmp3lame",
		"-b:a", "320k",
		"-f", "mp3",
		"-loglevel", "error",
		"pipe:1",
	)

	// Pipe yt-dlp stdout → FFmpeg stdin
	ytdlpOut, err := ytdlp.StdoutPipe()
	if err != nil {
		return fmt.Errorf("yt-dlp stdout pipe: %w", err)
	}
	ffmpeg.Stdin = ytdlpOut
	ffmpeg.Stdout = w

	// Start both processes
	if err := ytdlp.Start(); err != nil {
		return fmt.Errorf("yt-dlp start: %w", err)
	}
	if err := ffmpeg.Start(); err != nil {
		_ = ytdlp.Process.Kill()
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	// Wait for yt-dlp to finish (closes its stdout, signaling EOF to FFmpeg)
	ytdlpErr := ytdlp.Wait()

	// Wait for FFmpeg to finish processing
	ffmpegErr := ffmpeg.Wait()

	if ytdlpErr != nil {
		return fmt.Errorf("yt-dlp: %w", ytdlpErr)
	}
	if ffmpegErr != nil {
		return fmt.Errorf("ffmpeg: %w", ffmpegErr)
	}

	return nil
}

