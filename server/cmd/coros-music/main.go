package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/victor/coros-music/internal/pipeline"
	"github.com/victor/coros-music/internal/spotify"
	"github.com/victor/coros-music/internal/sync"
)

func main() {
	port := os.Getenv("COROS_PORT")
	if port == "" {
		port = "8080"
	}

	spotifyAuth := spotify.NewAuth()
	coordinator := sync.NewCoordinator()

	mux := http.NewServeMux()

	// Spotify (Client Credentials — no user login needed)
	mux.HandleFunc("GET /api/spotify/status", spotifyAuth.HandleStatus)
	mux.HandleFunc("GET /api/spotify/playlists", spotifyAuth.HandlePlaylists)
	mux.HandleFunc("GET /api/spotify/playlists/{id}", spotifyAuth.HandlePlaylistByID)
	mux.HandleFunc("GET /api/spotify/playlists/{id}/tracks", spotifyAuth.HandleTracks)

	// Sync streaming
	mux.HandleFunc("POST /api/sync/stream", coordinator.HandleStream)

	// Health check
	mux.HandleFunc("GET /api/health", handleHealth)

	// Static files (production)
	staticDir := os.Getenv("COROS_STATIC_DIR")
	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	log.Printf("CorosMusic server starting on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	ffmpegOk := pipeline.CheckBinary("ffmpeg")
	ytdlpOk := pipeline.CheckBinary("yt-dlp")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ffmpeg":%t,"ytdlp":%t,"spotify":true}`, ffmpegOk, ytdlpOk)
}

