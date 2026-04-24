package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

	certFile, keyFile, err := ensureTLSCertFiles()
	if err != nil {
		log.Fatalf("failed to prepare TLS certs: %v", err)
	}

	log.Printf("CorosMusic server starting on https://0.0.0.0:%s", port)
	if err := http.ListenAndServeTLS(":"+port, certFile, keyFile, mux); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	ffmpegOk := pipeline.CheckBinary("ffmpeg")
	ytdlpOk := pipeline.CheckBinary("yt-dlp")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ffmpeg":%t,"ytdlp":%t,"spotify":true}`, ffmpegOk, ytdlpOk)
}

func getDataDir() string {
	if dir := os.Getenv("COROS_DATA_DIR"); dir != "" {
		return dir
	}
	// Docker default with current compose volume mount.
	if _, err := os.Stat("/root/.coros-music"); err == nil {
		return "/root/.coros-music"
	}
	// Local fallback from repo root.
	return "coros-data"
}

func ensureTLSCertFiles() (certPath string, keyPath string, err error) {
	dataDir := getDataDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", "", fmt.Errorf("mkdir data dir: %w", err)
	}

	certPath = filepath.Join(dataDir, "cert.pem")
	keyPath = filepath.Join(dataDir, "key.pem")

	if fileExists(certPath) && fileExists(keyPath) {
		return certPath, keyPath, nil
	}

	certPEM, keyPEM, err := generateSelfSignedCertPEM()
	if err != nil {
		return "", "", err
	}

	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return "", "", fmt.Errorf("write cert.pem: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return "", "", fmt.Errorf("write key.pem: %w", err)
	}

	log.Printf("generated self-signed TLS certs in %s", dataDir)
	return certPath, keyPath, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func generateSelfSignedCertPEM() (certPEM []byte, keyPEM []byte, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("serial number: %w", err)
	}

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"CorosMusic"},
			CommonName:   "CorosMusic Local Dev",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           collectCertIPs(),
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}

func collectCertIPs() []net.IP {
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}

	seen := map[string]struct{}{}
	for _, ip := range ips {
		seen[ip.String()] = struct{}{}
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		if ip = ip.To4(); ip == nil {
			continue // keep cert small; IPv4 is enough for typical LAN setup
		}
		if _, ok := seen[ip.String()]; ok {
			continue
		}
		seen[ip.String()] = struct{}{}
		ips = append(ips, ip)
	}

	return ips
}

