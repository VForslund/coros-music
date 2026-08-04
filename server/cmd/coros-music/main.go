package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/victor/coros-music/internal"
	"github.com/victor/coros-music/internal/cert"
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

	mux.HandleFunc("GET /api/spotify/status", spotifyAuth.HandleStatus)
	mux.HandleFunc("GET /api/spotify/playlists", spotifyAuth.HandlePlaylists)
	mux.HandleFunc("GET /api/spotify/playlists/{id}", spotifyAuth.HandlePlaylistByID)
	mux.HandleFunc("GET /api/spotify/playlists/{id}/tracks", spotifyAuth.HandleTracks)

	mux.HandleFunc("POST /api/sync/stream", coordinator.HandleStream)

	mux.HandleFunc("GET /api/health", internal.HandleHealth)

	staticDir := os.Getenv("COROS_STATIC_DIR")
	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	certFile, keyFile, err := cert.GetTLSCertAndKey()
	if err != nil {
		log.Fatalf("failed to prepare TLS certs: %v", err)
	}

	log.Printf("CorosMusic server starting on https://0.0.0.0:%s", port)
	server := &http.Server{
		Addr:     ":" + port,
		Handler:  mux,
		ErrorLog: log.New(tlsNoiseFilterWriter{}, "", log.LstdFlags),
	}
	if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatal(err)
	}
}

type tlsNoiseFilterWriter struct{}

func (tlsNoiseFilterWriter) Write(p []byte) (int, error) {
	msg := string(p)
	// Suppress expected browser probe noise for self-signed certs.
	if strings.Contains(msg, "http: TLS handshake error") &&
		(strings.Contains(msg, "unknown certificate") || strings.Contains(msg, "bad certificate")) {
		return len(p), nil
	}
	return os.Stderr.Write(p)
}
