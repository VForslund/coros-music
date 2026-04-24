# ── Stage 1: Go backend ──
FROM golang:1.23-alpine AS backend
RUN apk add --no-cache ffmpeg python3 py3-pip && pip install yt-dlp --break-system-packages
WORKDIR /app
COPY server/ .
RUN go build -o coros-music ./cmd/coros-music

# ── Stage 2: Angular frontend ──
FROM node:22-alpine AS frontend
WORKDIR /app
COPY front_end/package.json front_end/package-lock.json ./
RUN npm ci
COPY front_end/ .
RUN npx ng build --configuration production

# ── Stage 3: Runtime ──
FROM alpine:3.20
RUN apk add --no-cache ffmpeg python3 py3-pip && pip install yt-dlp --break-system-packages
COPY --from=backend /app/coros-music /usr/local/bin/
COPY --from=frontend /app/dist/coros_music/browser /srv/static
ENV COROS_STATIC_DIR=/srv/static
EXPOSE 8080
CMD ["coros-music"]
