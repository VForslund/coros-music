package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func pickAudioEncoder(ctx context.Context, ffmpegPath string) string {
	if forced := os.Getenv("COROS_AUDIO_ENCODER"); forced != "" {
		return forced
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, "-hide_banner", "-encoders")
	out, err := cmd.Output()
	if err != nil {
		return "libmp3lame"
	}
	enc := string(out)
	if strings.Contains(enc, "libmp3lame") {
		return "libmp3lame"
	}
	if strings.Contains(enc, " mp3") {
		return "mp3"
	}
	return "libmp3lame"
}

func transcodeFromPipe(ctx context.Context, ytdlpPath, ffmpegPath, encoder, searchQuery string, w io.Writer) error {
	ytdlp := exec.CommandContext(ctx, ytdlpPath,
		"--no-playlist",
		"-f", "bestaudio",
		"-o", "-",
		fmt.Sprintf("ytsearch1:%s", searchQuery),
	)

	ffmpeg := exec.CommandContext(ctx, ffmpegPath,
		"-i", "pipe:0",
		"-vn",
		"-codec:a", encoder,
		"-b:a", "320k",
		"-f", "mp3",
		"-loglevel", "error",
		"pipe:1",
	)

	var ytdlpStderr bytes.Buffer
	var ffmpegStderr bytes.Buffer
	ytdlp.Stderr = &ytdlpStderr
	ffmpeg.Stderr = &ffmpegStderr

	ytdlpOut, err := ytdlp.StdoutPipe()
	if err != nil {
		return fmt.Errorf("yt-dlp stdout pipe: %w", err)
	}
	ffmpeg.Stdin = ytdlpOut
	ffmpeg.Stdout = w

	if err := ytdlp.Start(); err != nil {
		return fmt.Errorf("yt-dlp start: %w", err)
	}
	if err := ffmpeg.Start(); err != nil {
		_ = ytdlp.Process.Kill()
		return fmt.Errorf("ffmpeg start (encoder=%s): %w; ffmpeg_stderr=%s; ytdlp_stderr=%s",
			encoder, err, strings.TrimSpace(ffmpegStderr.String()), strings.TrimSpace(ytdlpStderr.String()))
	}

	ytdlpErr := ytdlp.Wait()
	ffmpegErr := ffmpeg.Wait()

	if ytdlpErr != nil {
		return fmt.Errorf("yt-dlp: %w; ytdlp_stderr=%s", ytdlpErr, strings.TrimSpace(ytdlpStderr.String()))
	}
	if ffmpegErr != nil {
		return fmt.Errorf("ffmpeg (encoder=%s): %w; ffmpeg_stderr=%s; ytdlp_stderr=%s",
			encoder, ffmpegErr, strings.TrimSpace(ffmpegStderr.String()), strings.TrimSpace(ytdlpStderr.String()))
	}

	return nil
}

func transcodeFromResolvedURL(ctx context.Context, ffmpegPath, encoder, searchQuery string, w io.Writer) error {
	url, err := Resolve(ctx, searchQuery)
	if err != nil {
		return err
	}

	ffmpeg := exec.CommandContext(ctx, ffmpegPath,
		"-i", url,
		"-vn",
		"-codec:a", encoder,
		"-b:a", "320k",
		"-f", "mp3",
		"-loglevel", "error",
		"pipe:1",
	)

	var ffmpegStderr bytes.Buffer
	ffmpeg.Stderr = &ffmpegStderr
	ffmpeg.Stdout = w

	if err := ffmpeg.Run(); err != nil {
		return fmt.Errorf("ffmpeg resolved-url (encoder=%s): %w; ffmpeg_stderr=%s", encoder, err, strings.TrimSpace(ffmpegStderr.String()))
	}
	return nil
}

// Transcode pipes yt-dlp audio through FFmpeg to produce a 320 kbps MP3 stream.
// If the piping path fails with invalid input, it retries via resolved direct URL.
func Transcode(ctx context.Context, searchQuery string, w io.Writer) error {
	ytdlpPath := os.Getenv("COROS_YTDLP_PATH")
	if ytdlpPath == "" {
		ytdlpPath = "yt-dlp"
	}
	ffmpegPath := os.Getenv("COROS_FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	encoder := pickAudioEncoder(ctx, ffmpegPath)

	if err := transcodeFromPipe(ctx, ytdlpPath, ffmpegPath, encoder, searchQuery, w); err == nil {
		return nil
	} else {
		msg := err.Error()
		// Known flaky case: ffmpeg cannot parse piped input from yt-dlp.
		if strings.Contains(msg, "Invalid data found when processing input") || strings.Contains(msg, "Error opening input file pipe:0") {
			if retryErr := transcodeFromResolvedURL(ctx, ffmpegPath, encoder, searchQuery, w); retryErr == nil {
				return nil
			} else {
				return fmt.Errorf("pipe_transcode_failed=%v; resolved_url_retry_failed=%v", err, retryErr)
			}
		}
		return err
	}
}
