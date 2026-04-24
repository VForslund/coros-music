package spotify

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	spotifyLib "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

var playlistIDPattern = regexp.MustCompile(`^[a-zA-Z0-9]{22}$`)

// Auth uses Client Credentials flow — no user login needed.
// Works for any public playlist. Only requires SPOTIFY_CLIENT_ID + SPOTIFY_CLIENT_SECRET.
type Auth struct {
	mu     sync.RWMutex
	client *spotifyLib.Client
}

func NewAuth() *Auth {
	a := &Auth{}
	a.authenticate()
	return a
}

func (a *Auth) authenticate() {
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		TokenURL:     spotifyauth.TokenURL,
	}
	ctx := context.Background()
	httpClient := config.Client(ctx)
	client := spotifyLib.New(httpClient)

	a.mu.Lock()
	a.client = client
	a.mu.Unlock()

	log.Println("Spotify client credentials authenticated")
}

func (a *Auth) getClient() *spotifyLib.Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

func isValidPlaylistID(id string) bool {
	return playlistIDPattern.MatchString(id)
}

func writeSpotifyError(w http.ResponseWriter, action string, err error) {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "404") || strings.Contains(msg, "not found"):
		http.Error(w, action+": playlist not found or not public", http.StatusNotFound)
	case strings.Contains(msg, "403") || strings.Contains(msg, "forbidden"):
		http.Error(w, action+": playlist is not accessible with app credentials", http.StatusForbidden)
	case strings.Contains(msg, "400") || strings.Contains(msg, "bad request"):
		http.Error(w, action+": invalid request", http.StatusBadRequest)
	default:
		log.Printf("spotify api error (%s): %v", action, err)
		http.Error(w, action+": spotify upstream error", http.StatusBadGateway)
	}
}

func (a *Auth) HandleStatus(w http.ResponseWriter, r *http.Request) {
	// Always authenticated with client credentials
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"authenticated":true}`))
}

func (a *Auth) HandlePlaylists(w http.ResponseWriter, r *http.Request) {
	// Client credentials can't list "current user's" playlists (no user context).
	// Instead, the frontend will let users paste a playlist URL/ID directly.
	// This endpoint now accepts a query param ?userId= to fetch a user's public playlists.
	client := a.getClient()
	ctx := r.Context()

	userID := r.URL.Query().Get("userId")
	if userID == "" {
		// Return empty — frontend should prompt for playlist URL instead
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
		return
	}

	playlists, err := client.GetPlaylistsForUser(ctx, userID, spotifyLib.Limit(50))
	if err != nil {
		http.Error(w, "failed to fetch playlists: "+err.Error(), http.StatusInternalServerError)
		return
	}

	type PlaylistJSON struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		TrackCount int    `json:"trackCount"`
		ImageURL   string `json:"imageURL"`
	}

	result := make([]PlaylistJSON, 0, len(playlists.Playlists))
	for _, p := range playlists.Playlists {
		img := ""
		if len(p.Images) > 0 {
			img = p.Images[0].URL
		}
		result = append(result, PlaylistJSON{
			ID:         string(p.ID),
			Name:       p.Name,
			TrackCount: int(p.Tracks.Total),
			ImageURL:   img,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandlePlaylistByID fetches a single playlist by ID (works for any public playlist).
func (a *Auth) HandlePlaylistByID(w http.ResponseWriter, r *http.Request) {
	client := a.getClient()
	if client == nil {
		http.Error(w, "spotify client not initialized", http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()
	playlistID := r.PathValue("id")
	if !isValidPlaylistID(playlistID) {
		http.Error(w, "invalid playlist id format", http.StatusBadRequest)
		return
	}

	playlist, err := client.GetPlaylist(ctx, spotifyLib.ID(playlistID))
	if err != nil {
		writeSpotifyError(w, "failed to fetch playlist", err)
		return
	}

	type PlaylistJSON struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		TrackCount int    `json:"trackCount"`
		ImageURL   string `json:"imageURL"`
	}

	img := ""
	if len(playlist.Images) > 0 {
		img = playlist.Images[0].URL
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PlaylistJSON{
		ID:         string(playlist.ID),
		Name:       playlist.Name,
		TrackCount: int(playlist.Tracks.Total),
		ImageURL:   img,
	})
}

func (a *Auth) HandleTracks(w http.ResponseWriter, r *http.Request) {
	client := a.getClient()
	if client == nil {
		http.Error(w, "spotify client not initialized", http.StatusServiceUnavailable)
		return
	}
	playlistID := r.PathValue("id")
	if !isValidPlaylistID(playlistID) {
		http.Error(w, "invalid playlist id format", http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	type TrackJSON struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Artist       string `json:"artist"`
		Album        string `json:"album"`
		DurationMs   int    `json:"durationMs"`
		SyncFilename string `json:"syncFilename"`
	}

	var allTracks []TrackJSON
	offset := 0
	limit := 100

	for {
		tracks, err := client.GetPlaylistTracks(ctx, spotifyLib.ID(playlistID),
			spotifyLib.Limit(limit), spotifyLib.Offset(offset))
		if err != nil {
			writeSpotifyError(w, "failed to fetch tracks", err)
			return
		}

		for _, item := range tracks.Tracks {
			t := item.Track
			artist := ""
			if len(t.Artists) > 0 {
				artist = t.Artists[0].Name
			}
			allTracks = append(allTracks, TrackJSON{
				ID:           string(t.ID),
				Title:        t.Name,
				Artist:       artist,
				Album:        t.Album.Name,
				DurationMs:   int(t.Duration),
				SyncFilename: sanitizeFilename(string(t.ID), artist, t.Name),
			})
		}

		if len(tracks.Tracks) < limit {
			break
		}
		offset += limit
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allTracks)
}
