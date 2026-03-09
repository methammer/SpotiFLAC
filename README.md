  # SpotiFLAC Web

A self-hosted web app to download Spotify tracks in true FLAC from Tidal, Qobuz, Amazon Music & Deezer — no account required.

> **Based on [SpotiFLAC](https://github.com/afkarxyz/SpotiFLAC) by afkarxyz** — rewritten as a web server with multi-user support and Jellyfin integration.

## Features

- 🎵 Download Spotify tracks, albums and playlists as FLAC
- 👥 **Multi-user** — authentication via your Jellyfin server
- 📋 **Watchlists** — auto-sync Spotify playlists at configurable intervals
- 🎬 **Jellyfin integration** — generates M3U8 playlist files automatically
- 🔄 **Smart sync** — detects new tracks, optionally deletes removed ones (with multi-playlist protection)
- 📊 Real-time download queue with progress, speed and size
- 🗂️ File browser, audio converter, audio analysis
- 🧹 Automatic BoltDB cleanup (deduplication every 24h)
- 🐳 Docker-first deployment

## Screenshots

> *(add your screenshots here)*

## Quick Start

### 1. Prerequisites

- Docker + Docker Compose
- A running [Jellyfin](https://jellyfin.org) instance (used for authentication)
- FFmpeg (bundled in the Docker image)

### 2. Deploy

```bash
git clone https://github.com/YOURUSER/spotiflac-web
cd spotiflac-web
cp docker-compose.example.yaml docker-compose.yaml
# Edit docker-compose.yaml with your paths and settings
docker compose up -d
```

### 3. Configure `docker-compose.yaml`

```yaml
services:
  spotiflac:
    image: ghcr.io/YOURUSER/spotiflac:latest
    container_name: spotiflac
    restart: unless-stopped
    ports:
      - "6890:6890"
    environment:
      - JELLYFIN_URL=http://your-jellyfin-host:8096
      - JWT_SECRET=change-me-to-a-random-32-char-string
    volumes:
      - /path/to/music:/home/nonroot/Music
      - /path/to/config:/home/nonroot/.SpotiFLAC
```

### 4. Access

Open `http://your-server:6890` and log in with your Jellyfin credentials.

> All Jellyfin users can log in. Each user has their own watchlists, download queue and settings.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JELLYFIN_URL` | `http://localhost:8096` | URL of your Jellyfin instance |
| `JWT_SECRET` | *(insecure default)* | Secret key for JWT signing — **change in production** |

## Reverse Proxy (Nginx example)

```nginx
location / {
    proxy_pass http://localhost:6890;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_read_timeout 0;
}
```

## Architecture

```
Browser → /auth/login → Jellyfin auth → JWT (24h)
Browser → /api/rpc + Bearer token → handlers (per-user filtered)
                                  → BoltDB (jobs, watchlists, users)
                                  → Download workers
```

**Data isolation per user:**
- Watchlists
- Download queue & history  
- Settings (quality, download path, templates)

## Building from Source

```bash
# Requirements: Go 1.21+, Bun
cd frontend && bun install && bun run build
cd ..
go build -o spotiflac .

# Or with Docker
docker build -t spotiflac:local .
```

## Data Storage

All data is stored in the config volume:

| File | Description |
|------|-------------|
| `jobs.db` | Download jobs, watchlists, users (BoltDB) |
| `history.db` | Download history |
| `config.json` | Global settings |

## Differences from original SpotiFLAC

| Feature | Original | Web |
|---------|----------|-----|
| Interface | Desktop (Wails) | Web browser |
| Auth | None | Jellyfin login |
| Multi-user | ❌ | ✅ |
| Watchlists | ✅ | ✅ |
| M3U8 Jellyfin | ❌ | ✅ |
| Docker | ❌ | ✅ |
| Self-hosted | ❌ | ✅ |

## Disclaimer

This project is for **educational and private use only**.

**SpotiFLAC Web** is not affiliated with Spotify, Tidal, Qobuz, Amazon Music, Deezer, Jellyfin or any other service. You are solely responsible for ensuring your use complies with your local laws and the Terms of Service of the respective platforms.

## Credits

- [afkarxyz/SpotiFLAC](https://github.com/afkarxyz/SpotiFLAC) — original project
- [MusicBrainz](https://musicbrainz.org) · [LRCLIB](https://lrclib.net) · [Song.link](https://song.link) · [hifi-api](https://github.com/binimum/hifi-api)
