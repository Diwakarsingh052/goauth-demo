# Deployment Guide — Digital Ocean Droplet

**Domain:** `learncodeacademy.com`
**Droplet IP:** `198.199.78.177`

## Architecture

```
Internet
  │
  ▼
┌─────────────────────────────────────────────┐
│  Caddy  (ports 80 / 443)                    │  ← automatic HTTPS
│  ┌───────────────────────────────────────┐  │
│  │ learncodeacademy.com  →  web (:8081)  │  │
│  └───────────────────────────────────────┘  │
│                                             │
│  Web ──HTTP──► API (:8080) ──SQL──► MySQL   │
└─────────────────────────────────────────────┘
```

| Service | Internal Port | Publicly Exposed | Purpose                     |
| ------- | ------------- | ---------------- | --------------------------- |
| Caddy   | 80 / 443      | Yes              | Reverse proxy, auto TLS     |
| Web     | 8081          | No (via Caddy)   | HTML frontend (Go)          |
| API     | 8080          | No               | REST API (Go, internal)     |
| MySQL   | 3306          | No               | Database                    |

---

## Step 1 — Point DNS to the Droplet

Go to your domain registrar (wherever you bought `learncodeacademy.com`) and add an **A record**:

| Type | Name | Value             | TTL  |
| ---- | ---- | ----------------- | ---- |
| A    | @    | `198.199.78.177`  | 3600 |

> **`@`** means the root domain (`learncodeacademy.com`).

### Verify DNS propagation

After saving, wait a few minutes then check:

```bash
# From your local machine
nslookup learncodeacademy.com
```

You should see `198.199.78.177` in the response. If not, wait and try again — DNS can take up to 30 minutes to propagate (usually faster).

---

## Step 2 — SSH into the Droplet

```bash
ssh root@198.199.78.177
```

---

## Step 3 — Install Docker

```bash
curl -fsSL https://get.docker.com | sh

# Verify
docker compose version
```

---

## Step 4 — Configure Firewall

```bash
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

---

## Step 5 — Clone the Repository

```bash
cd /opt
git clone <YOUR_REPO_URL> challange-go-cyaz
cd challange-go-cyaz
```

---

## Step 6 — Create the `.env` File

```bash
nano .env
```

Paste the following (replace the placeholder values):

```env
# ── Database ────────────────────────────────────────────
DB_PASSWORD=<STRONG_RANDOM_PASSWORD>
DB_NAME=challange_go

# ── Secrets ─────────────────────────────────────────────
JWT_SECRET=<RANDOM_64_CHAR_STRING>
SESSION_SECRET=<RANDOM_64_CHAR_STRING>

# ── Google OAuth ────────────────────────────────────────
GOOGLE_CLIENT_ID=<your-google-client-id>
GOOGLE_CLIENT_SECRET=<your-google-client-secret>
GOOGLE_REDIRECT_URL=https://learncodeacademy.com/auth/google/callback
```

Generate random secrets:

```bash
openssl rand -hex 32   # run twice — use for JWT_SECRET and SESSION_SECRET
openssl rand -hex 16   # use for DB_PASSWORD
```

---

## Step 7 — Configure Google OAuth

1. Go to [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials)
2. Click on your **OAuth 2.0 Client ID** (or create one)
3. Under **Authorized JavaScript origins**, add:
   ```
   https://learncodeacademy.com
   ```
4. Under **Authorized redirect URIs**, add:
   ```
   https://learncodeacademy.com/auth/google/callback
   ```
5. Click **Save**
6. Copy the **Client ID** and **Client Secret** into your `.env` file (Step 6)

---

## Step 8 — Deploy

```bash
cd /opt/challange-go-cyaz
docker compose -f docker-compose.prod.yml up -d --build
```

Caddy will automatically obtain a TLS certificate from Let's Encrypt within seconds.

---

## Step 9 — Verify

```bash
# All 4 containers should be "running"
docker compose -f docker-compose.prod.yml ps

# Check for errors
docker compose -f docker-compose.prod.yml logs -f

# Test HTTPS
curl -I https://learncodeacademy.com
```

You should see a `200 OK` or `303 See Other` redirect to `/login`.
Open **https://learncodeacademy.com** in your browser to confirm.

---

## Common Operations

### View Logs

```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Single service
docker compose -f docker-compose.prod.yml logs -f web
docker compose -f docker-compose.prod.yml logs -f caddy
```

### Restart Services

```bash
docker compose -f docker-compose.prod.yml restart
```

### Redeploy After Code Changes

```bash
cd /opt/challange-go-cyaz
git pull origin main
docker compose -f docker-compose.prod.yml up -d --build
```

### Stop Everything

```bash
docker compose -f docker-compose.prod.yml down
```

### Stop and Wipe Data (destructive)

```bash
docker compose -f docker-compose.prod.yml down -v
```

---

## Caddy Details

The `Caddyfile` lives in the repo root. Key points:

- **Automatic HTTPS** — Caddy provisions Let's Encrypt certs automatically. No manual cert setup needed.
- **HTTP → HTTPS redirect** — Caddy redirects all HTTP traffic to HTTPS by default.
- **HTTP/3** — Enabled by exposing UDP port 443.
- **Static asset caching** — Files under `/static/*` get a 30-day `Cache-Control` header.
- **Security headers** — `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`.
- **API stays internal** — The REST API is only reachable by the web container over the Docker network.

To reload Caddy config without downtime:

```bash
docker compose -f docker-compose.prod.yml exec caddy caddy reload --config /etc/caddy/Caddyfile
```

---

## Database Backups

### Manual Backup

```bash
docker compose -f docker-compose.prod.yml exec mysql \
  mysqldump -u root -p"${DB_PASSWORD}" challange_go > backup_$(date +%Y%m%d).sql
```

### Automated Daily Backups (cron)

```bash
mkdir -p /opt/backups

crontab -e

# Add this line (runs daily at 2 AM)
0 2 * * * cd /opt/challange-go-cyaz && docker compose -f docker-compose.prod.yml exec -T mysql mysqldump -u root -p"$(grep DB_PASSWORD .env | cut -d= -f2)" challange_go | gzip > /opt/backups/db_$(date +\%Y\%m\%d).sql.gz
```

---

## Monitoring

### Basic Health Check

```bash
curl -sf https://learncodeacademy.com > /dev/null && echo "UP" || echo "DOWN"
```

### Resource Usage

```bash
docker stats --no-stream
```

### Disk Space

```bash
df -h
docker system df
```

### Clean Up Unused Docker Resources

```bash
docker system prune -af --volumes
```

---

## File Overview

| File                      | Purpose                                         |
| ------------------------- | ----------------------------------------------- |
| `Caddyfile`               | Caddy reverse proxy configuration               |
| `docker-compose.prod.yml` | Production Docker Compose (Caddy + app + MySQL)  |
| `docker-compose.yml`      | Development Docker Compose (no Caddy)            |
| `Dockerfile`              | Multi-stage build for API and Web binaries       |
| `.env`                    | Environment variables (not committed to git)     |

---

## Troubleshooting

| Problem                          | Fix                                                                                  |
| -------------------------------- | ------------------------------------------------------------------------------------ |
| Caddy can't get TLS certificate  | Ensure DNS A record points to `198.199.78.177` and ports 80 + 443 are open           |
| `502 Bad Gateway`                | Check if `web` / `api` containers are running: `docker compose -f docker-compose.prod.yml ps` |
| Database connection refused      | Wait for MySQL healthcheck; check `docker compose -f docker-compose.prod.yml logs mysql`       |
| Google OAuth redirect mismatch   | Verify Google Console redirect URI is exactly `https://learncodeacademy.com/auth/google/callback` |
| Out of disk space                | Run `docker system prune -af` and check `/var/lib/docker`                             |
| Containers keep restarting       | Check logs: `docker compose -f docker-compose.prod.yml logs <service>`                |
| Port 80 already in use           | Stop any existing web server: `systemctl stop nginx` or `systemctl stop apache2`      |