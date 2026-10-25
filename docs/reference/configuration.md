# Configuration

Reference of every setting. Radiopath is configured only through environment variables; for local development
`make dev` copies `.env.example` to `.env` (gitignored) and loads it, in Kubernetes the same variables are set on
the Deployment, secrets via `secretKeyRef`.

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | required | PostgreSQL URL, e.g. `postgres://user:pw@host:5432/radiopath` |
| `RADIOPATH_DEM_DIR` | | Directory with `.hgt` tiles (read-only volume in Kubernetes). Required unless the S3 source is configured |
| `RADIOPATH_DEM_S3_ENDPOINT` | | S3-compatible endpoint, e.g. `https://s3.example.org`. Enables the S3 source together with the bucket |
| `RADIOPATH_DEM_S3_BUCKET` | | Bucket holding the `.hgt` tiles |
| `RADIOPATH_DEM_S3_PREFIX` | empty | Key prefix, e.g. `srtm1/` for keys like `srtm1/N47E009.hgt` |
| `RADIOPATH_DEM_S3_REGION` | `us-east-1` | SigV4 region; keep the default for MinIO and Ceph |
| `RADIOPATH_DEM_S3_ACCESS_KEY` | | Access key. Empty: anonymous requests (public bucket) |
| `RADIOPATH_DEM_S3_SECRET_KEY` | | Secret key, from a Kubernetes Secret |
| `RADIOPATH_DEM_CACHE_TILES` | `8` | Tiles kept in memory; a 1" tile is 26 MB (also the size of the canopy cache, 13 MB per tile) |
| `RADIOPATH_CANOPY_DIR` | | Directory with `.chm` canopy height tiles. Empty: no vegetation model |
| `RADIOPATH_CANOPY_S3_PREFIX` | | Key prefix of the `.chm` tiles in the DEM bucket, e.g. `canopy1/`. Takes precedence over the directory |
| `RADIOPATH_LISTEN` | `:8080` | Listen address |
| `RADIOPATH_METRICS_LISTEN` | `:9090` | Listen address of the Prometheus endpoint (`/metrics`). A second listener on purpose, so it is never reachable through the ingress. Empty: no metrics |
| `RADIOPATH_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. Logs are always JSON lines on stderr |
| `RADIOPATH_WORKERS` | `1` | Coverage workers in this process; `0` for web-only pods, `1` on compute pods |
| `RADIOPATH_TRUST_PROXY` | `false` | Take the client IP for rate limiting from the last `X-Forwarded-For` hop. Set `true` behind your ingress only |
| `RADIOPATH_REGISTRATION` | `true` | Self-service account creation at `/register`; `false` hides it and only the CLI and the admin panel create users. Needs `RADIOPATH_SMTP_HOST`, new accounts confirm their address by mail |
| `RADIOPATH_SMTP_HOST` | empty | Submission server for confirmation and reset mails (STARTTLS required, PLAIN auth when a user is set). Empty: no registration and no "Forgot password" |
| `RADIOPATH_SMTP_PORT` | `587` | Port of that server |
| `RADIOPATH_SMTP_STARTTLS` | `true` | `false` sends in the clear: only for a relay on localhost or the e2e mail catcher; Go refuses PLAIN auth without TLS anyway, except to localhost |
| `RADIOPATH_SMTP_USER` | empty | Login for the submission server |
| `RADIOPATH_SMTP_PASSWORD` | empty | Its password, from a Kubernetes Secret |
| `RADIOPATH_MAIL_FROM` | | Sender address, required with the host |
| `RADIOPATH_BASE_URL` | | Public address of the app, e.g. `https://radiopath.example.org`; the links in mails and the share links start with it. Required with the SMTP host; without it the share panel shows a path instead of a full URL |
| `RADIOPATH_ADMIN_TOKEN` | empty | Token for the admin panel at `/admin`, at least 32 characters. Empty: no admin panel |
| `RADIOPATH_REDIS_URL` | empty | Redis URL for the map tile cache, e.g. `redis://redis:6379/0`. Empty: tiles are proxied but not cached |
| `RADIOPATH_TILE_URL` | `https://tile.openstreetmap.org/{z}/{x}/{y}.png` | Upstream tile template (`{z}`, `{x}`, `{y}`, optional `{s}`) |
| `RADIOPATH_TILE_TTL` | `720h` | Cache lifetime of a tile, in Redis and in the browser |
| `RADIOPATH_TILE_MISSES_PER_IP` | `300` | Tiles one client IP may fetch upstream per minute; `0` disables the limit |

Migrations are embedded and applied at startup.
