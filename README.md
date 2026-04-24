# CorosMusic

Spotify playlist sync tool for COROS watches.

- Frontend: Angular 21 (`/` on `http://localhost:4200` in dev)
- Backend: Go API (`/api/*` on `http://localhost:8080`)
- Docker: single container serves both static UI and API on `http://localhost:8080`
- Pipeline tools: `yt-dlp` + `ffmpeg`

## 1) Run in Dev Mode

From the project root (`coros_music`), run backend and frontend in separate terminals.

### What you need for local development

- Node.js 22+
- npm 11+
- Go 1.23+
- `ffmpeg` available in `PATH`
- `yt-dlp` available in `PATH`

Optional checks:

```bash
ffmpeg -version
yt-dlp --version
go version
node -v
npm -v
```

### Backend

```bash
cd ./server
SPOTIFY_CLIENT_ID="your_client_id" SPOTIFY_CLIENT_SECRET="your_client_secret" go run ./cmd/coros-music
```

Optional backend env vars:

- `COROS_PORT` (default: `8080`)
- `COROS_WORKERS` (default: `4`)
- `COROS_FFMPEG_PATH` (default: `ffmpeg`)
- `COROS_YTDLP_PATH` (default: `yt-dlp`)

### Frontend

```bash
cd ./front_end
npm install
npx ng serve --proxy-config proxy.conf.json
```

The Angular proxy forwards `/api/*` to `http://localhost:8080` (`front_end/proxy.conf.json`).

Open:

- `http://localhost:4200` (app)
- `http://localhost:8080/api/health` (backend health)

## 2) Docker

Use the checked-in `Dockerfile` to build one image that contains:

- the Go backend binary
- the Angular production build
- `ffmpeg` and `yt-dlp` installed in the runtime container

### What you need for Docker

- Docker 24+
- A `./.env` file with Spotify credentials before you start Compose

### Step 1: Build the image

```bash
cd .
docker build -t coros-music:local .
```

### Step 2: Create `.env`

Create `./.env`:

```dotenv
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_CLIENT_SECRET=your_client_secret
```

### Step 3: Start with Compose

A `docker-compose.yml` is already included in the repo.

```bash
cd .
docker compose up --build
```

Then open `http://localhost:8080`.

## 3) Onboarding: First-Time Use

### Spotify app setup

What you need for onboarding:

- a Spotify developer app
- your Spotify **Client ID** and **Client Secret**
- a Chromium-based browser for watch folder access
- your COROS watch mounted so you can pick its root folder

1. Open the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard).
2. Sign in and create an app.
3. In your app settings, copy these credentials:
   - **Client ID**
   - **Client Secret**
4. No redirect URI is needed — CorosMusic uses the Client Credentials flow.
5. Put the credentials into either:
   - your shell environment for local dev, or
   - `./.env` for Docker Compose

Example `.env`:

```dotenv
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_CLIENT_SECRET=your_client_secret
```

### First run

1. Start the app (dev mode on `http://localhost:4200` or Docker on `http://localhost:8080`).
2. Open the corresponding URL in your browser.
3. Paste a Spotify playlist URL (e.g. `https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M`) into the input field and click **+ Add**.
4. The playlist and its tracks will appear.
5. Connect your COROS watch by clicking **Connect Watch** and selecting the watch root folder.
6. Ensure the selected folder contains a `Music` directory.
7. Click **▶ Sync** to start downloading and transferring tracks.

### Browser and device notes

- Use a Chromium-based browser (Chrome/Edge) for `showDirectoryPicker` support.
- Any public Spotify playlist can be synced — no Spotify login or consent screen required.
- Keep the watch mounted during sync.

## Useful Commands

```bash
# frontend unit tests
cd ./front_end
npm test

# backend run
cd ./server
go run ./cmd/coros-music
```
