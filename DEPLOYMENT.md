# Deployment Guide — Digital Ocean Droplet

## Architecture

```
Internet
  │
  ▼
┌──────────────────────────────────┐
│  Caddy  (ports 80 / 443)        │  ← automatic HTTPS via Let's Encrypt
│  ┌────────────────────────────┐  │
│  │ yourdomain.com      → web  │  │  ← Go web frontend  (port 8081)
│  │ api.yourdomain.com  → api  │  │  ← Go REST API      (port 8080)
│  └────────────────────────────┘  │
│                                  │
│  Web ──HTTP──► API ──SQL──► MySQL│
└──────────────────────────────────┘
```

| Service | Internal Port | Publicly Exposed | Purpose                     |
| ------- | ------------- | ---------------- | --------------------------- |
| Caddy   | 80 / 443      | Yes              | Reverse proxy, TLS          |
| Web     | 8081          | No (via Caddy)   | HTML frontend (Go)          |
| API     | 8080          | No (via Caddy)   | REST API (Go)               |
| MySQL   | 3306          | No               | Database                    |

---

## Prerequisites

- A **Digital Ocean Droplet** (Ubuntu 22.04+ recommended, minimum 1 GB RAM / 1 vCPU)
- A **domain name** with DNS pointed to the droplet's IP
- **Docker** and **Docker Compose** installed on the droplet

---

## 1. Droplet Setup

### 1.1 Create the Droplet

1. Log in to [DigitalOcean](https://cloud.digitalocean.com)
2. Create a new Droplet:
   - **Image:** Ubuntu 22.04 LTS
   - **Plan:** Basic — $6/mo (1 vCPU, 1 GB RAM) works for light traffic; pick 2 GB+ for production
   - **Datacenter:** Choose the region closest to your users
   - **Authentication:** SSH key (recommended)
3. Note the Droplet's public IP address

### 1.2 Point DNS

Add two **A records** to your domain's DNS:

| Type | Name  | Value              | TTL  |
| ---- | ----- | ------------------ | ---- |
| A    | @     | `<DROPLET_IP>`     | 3600 |
| A    | api   | `<DROPLET_IP>`     | 3600 |

> If you don't need the API exposed publicly, skip the `api` subdomain record and comment out the `api.{$DOMAIN}` block in `Caddyfile`.

### 1.3 Install Docker

```bash
# SSH into the droplet
ssh root@<DROPLET_IP>

# Install Docker
curl -fsSL https://get.docker.com | sh

# Install Docker Compose plugin (included in recent Docker installs)
docker compose version   # verify it works
```

### 1.4 Configure Firewall

```bash
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

---

## 2. Deploy the Application

### 2.1 Clone the Repository

```bash
cd /opt
git clone <YOUR_REPO_URL> challange-go-cyaz
cd challange-go-cyaz
```

### 2.2 Create the `.env` File

```bash
cp /dev/null .env
```

Populate `.env` with the following (replace placeholder values):

```env
# ── Domain ──────────────────────────────────────────────
DOMAIN=yourdomain.com

# ── Database ────────────────────────────────────────────
DB_PASSWORD=<STRONG_RANDOM_PASSWORD>
DB_NAME=challange_go

# ── Secrets ─────────────────────────────────────────────
JWT_SECRET=<RANDOM_64_CHAR_STRING>
SESSION_SECRET=<RANDOM_64_CHAR_STRING>

# ── Google OAuth (optional) ─────────────────────────────
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=https://yourdomain.com/auth/google/callback
```

Generate random secrets:

```bash
openssl rand -hex 32   # run twice, use for JWT_SECRET and SESSION_SECRET
openssl rand -hex 16   # use for DB_PASSWORD
```

### 2.3 Start Everything

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

Caddy will automatically obtain TLS certificates from Let's Encrypt within seconds.

### 2.4 Verify

```bash
# Check all containers are running
docker compose -f docker-compose.prod.yml ps

# Check logs
docker compose -f docker-compose.prod.yml logs -f

# Test HTTPS
curl -I https://yourdomain.com
```

You should see a `200 OK` or `303 See Other` redirect to `/login`.

---

## 3. Common Operations

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

## 4. Caddy Configuration

The `Caddyfile` lives in the repo root. Key points:

- **Automatic HTTPS** — Caddy provisions Let's Encrypt certs automatically. No manual cert setup.
- **HTTP/3** — Enabled by exposing UDP port 443.
- **Static asset caching** — Files under `/static/*` get a 30-day `Cache-Control` header.
- **Security headers** — `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` are set by default.
- **API subdomain** — `api.yourdomain.com` proxies to the REST API. Remove this block if you want the API internal-only.

To reload Caddy config without downtime:

```bash
docker compose -f docker-compose.prod.yml exec caddy caddy reload --config /etc/caddy/Caddyfile
```

---

## 5. SSL / TLS Details

Caddy handles everything automatically:

- Obtains certificates from Let's Encrypt (or ZeroSSL as fallback)
- Renews certificates before expiry
- Redirects HTTP → HTTPS
- Supports HTTP/2 and HTTP/3 out of the box

**Requirements for automatic HTTPS:**
1. DNS A records must point to the droplet IP
2. Ports 80 and 443 must be open (for ACME challenge)
3. The `DOMAIN` environment variable must be set

---

## 6. Database Backups

### Manual Backup

```bash
docker compose -f docker-compose.prod.yml exec mysql \
  mysqldump -u root -p"${DB_PASSWORD}" challange_go > backup_$(date +%Y%m%d).sql
```

### Automated Daily Backups (cron)

```bash
# Edit crontab
crontab -e

# Add this line (runs daily at 2 AM)
0 2 * * * cd /opt/challange-go-cyaz && docker compose -f docker-compose.prod.yml exec -T mysql mysqldump -u root -p"$(grep DB_PASSWORD .env | cut -d= -f2)" challange_go | gzip > /opt/backups/db_$(date +\%Y\%m\%d).sql.gz
```

```bash
mkdir -p /opt/backups
```

---

## 7. Monitoring

### Basic Health Check

```bash
curl -sf https://yourdomain.com > /dev/null && echo "UP" || echo "DOWN"
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

## 8. Updating Google OAuth for Production

When moving from localhost to production, update the Google Cloud Console:

1. Go to [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials)
2. Edit your OAuth 2.0 Client ID
3. Add **Authorized redirect URIs**:
   ```
   https://yourdomain.com/auth/google/callback
   ```
4. Update `.env`:
   ```env
   GOOGLE_CLIENT_ID=your-client-id
   GOOGLE_CLIENT_SECRET=your-client-secret
   GOOGLE_REDIRECT_URL=https://yourdomain.com/auth/google/callback
   ```
5. Restart:
   ```bash
   docker compose -f docker-compose.prod.yml up -d
   ```

---

## 9. File Overview

| File                      | Purpose                                         |
| ------------------------- | ----------------------------------------------- |
| `Caddyfile`               | Caddy reverse proxy configuration               |
| `docker-compose.prod.yml` | Production Docker Compose (Caddy + app + MySQL)  |
| `docker-compose.yml`      | Development Docker Compose (no Caddy)            |
| `Dockerfile`              | Multi-stage build for API and Web binaries       |
| `.env`                    | Environment variables (not committed to git)     |

---

## 10. Troubleshooting

| Problem                         | Fix                                                                                  |
| -------------------------------- | ------------------------------------------------------------------------------------ |
| Caddy can't get certificates     | Ensure DNS points to the droplet and ports 80/443 are open                           |
| `502 Bad Gateway`                | Check if `web` / `api` containers are running: `docker compose -f docker-compose.prod.yml ps` |
| Database connection refused      | Wait for MySQL healthcheck; check `docker compose -f docker-compose.prod.yml logs mysql`       |
| Google OAuth not working         | Verify `GOOGLE_REDIRECT_URL` matches Google Console and uses `https://`               |
| Out of disk space                | Run `docker system prune -af` and check `/var/lib/docker`                             |
| Containers keep restarting       | Check logs: `docker compose -f docker-compose.prod.yml logs <service>`                |