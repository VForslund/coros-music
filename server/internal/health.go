package internal

import (
	"fmt"
	"net/http"

	"github.com/victor/coros-music/internal/pipeline"
)


func HandleHealth(w http.ResponseWriter, r *http.Request) {
	ffmpegOk := pipeline.CheckBinary("ffmpeg")
	ytdlpOk := pipeline.CheckBinary("yt-dlp")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ffmpeg":%t,"ytdlp":%t,"spotify":true}`, ffmpegOk, ytdlpOk)
}
