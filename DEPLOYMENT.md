# Deployment Guide — Digital Ocean Droplet (IP-based)

## Architecture

```
Internet
  │
  ▼
┌──────────────────────────────────┐
│  Caddy  (port 80)               │  ← HTTP reverse proxy
│  ┌────────────────────────────┐  │
│  │ http://<DROPLET_IP>  → web │  │  ← Go web frontend  (port 8081)
│  └────────────────────────────┘  │
│                                  │
│  Web ──HTTP──► API ──SQL──► MySQL│
│              (8080)       (3306) │
└──────────────────────────────────┘
```

| Service | Internal Port | Publicly Exposed | Purpose                     |
| ------- | ------------- | ---------------- | --------------------------- |
| Caddy   | 80            | Yes              | Reverse proxy               |
| Web     | 8081          | No (via Caddy)   | HTML frontend (Go)          |
| API     | 8080          | No               | REST API (Go, internal)     |
| MySQL   | 3306          | No               | Database                    |

> **Note:** This setup serves over plain HTTP. Let's Encrypt cannot issue certificates for bare IP addresses. When you get a domain, see [Upgrading to HTTPS](#8-upgrading-to-https-when-you-get-a-domain) below.

---

## Prerequisites

- A **Digital Ocean Droplet** (Ubuntu 22.04+ recommended, minimum 1 GB RAM / 1 vCPU)
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

### 1.2 Install Docker

```bash
# SSH into the droplet
ssh root@<DROPLET_IP>

# Install Docker
curl -fsSL https://get.docker.com | sh

# Install Docker Compose plugin (included in recent Docker installs)
docker compose version   # verify it works
```

### 1.3 Configure Firewall

```bash
ufw allow OpenSSH
ufw allow 80/tcp
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
# ── Database ────────────────────────────────────────────
DB_PASSWORD=<STRONG_RANDOM_PASSWORD>
DB_NAME=challange_go

# ── Secrets ─────────────────────────────────────────────
JWT_SECRET=<RANDOM_64_CHAR_STRING>
SESSION_SECRET=<RANDOM_64_CHAR_STRING>

# ── Google OAuth (optional) ─────────────────────────────
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=http://<DROPLET_IP>/auth/google/callback
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

### 2.4 Verify

```bash
# Check all containers are running
docker compose -f docker-compose.prod.yml ps

# Check logs
docker compose -f docker-compose.prod.yml logs -f

# Test it
curl -I http://<DROPLET_IP>
```

You should see a `200 OK` or `303 See Other` redirect to `/login`.
Open `http://<DROPLET_IP>` in your browser to confirm.

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

- **HTTP only** — Serving on port 80 since there's no domain for TLS.
- **Static asset caching** — Files under `/static/*` get a 30-day `Cache-Control` header.
- **Security headers** — `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` are set by default.
- **API stays internal** — The REST API is only reachable by the web container over the Docker network. Not exposed to the internet.

To reload Caddy config without downtime:

```bash
docker compose -f docker-compose.prod.yml exec caddy caddy reload --config /etc/caddy/Caddyfile
```

---

## 5. Database Backups

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

## 6. Monitoring

### Basic Health Check

```bash
curl -sf http://<DROPLET_IP> > /dev/null && echo "UP" || echo "DOWN"
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

## 7. Google OAuth Setup

Update the Google Cloud Console for your droplet IP:

1. Go to [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials)
2. Edit your OAuth 2.0 Client ID
3. Add **Authorized redirect URIs**:
   ```
   http://<DROPLET_IP>/auth/google/callback
   ```
4. Update `.env`:
   ```env
   GOOGLE_CLIENT_ID=your-client-id
   GOOGLE_CLIENT_SECRET=your-client-secret
   GOOGLE_REDIRECT_URL=http://<DROPLET_IP>/auth/google/callback
   ```
5. Restart:
   ```bash
   docker compose -f docker-compose.prod.yml up -d
   ```

> **Note:** Google OAuth requires HTTPS for production redirect URIs (except `localhost`). With an IP-only setup, Google may reject non-localhost HTTP callbacks. You may need a domain + HTTPS before Google OAuth works in production.

---

## 8. Upgrading to HTTPS (When You Get a Domain)

When you're ready to add a domain:

1. **Point DNS** — Add an A record for your domain pointing to the droplet IP.

2. **Update the Caddyfile** — Replace `:80` with your domain:
   ```
   yourdomain.com {
       reverse_proxy web:8081
       # ... rest stays the same
   }
   ```

3. **Update `docker-compose.prod.yml`** — Add HTTPS ports to the caddy service:
   ```yaml
   ports:
     - "80:80"
     - "443:443"
     - "443:443/udp"   # HTTP/3
   ```

4. **Open port 443 on the firewall:**
   ```bash
   ufw allow 443/tcp
   ```

5. **Update `.env`** — Change the Google redirect URL to `https://`:
   ```env
   GOOGLE_REDIRECT_URL=https://yourdomain.com/auth/google/callback
   ```

6. **Redeploy:**
   ```bash
   docker compose -f docker-compose.prod.yml up -d --build
   ```

Caddy will automatically get a Let's Encrypt certificate and redirect HTTP → HTTPS.

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
| `502 Bad Gateway`                | Check if `web` / `api` containers are running: `docker compose -f docker-compose.prod.yml ps` |
| Database connection refused      | Wait for MySQL healthcheck; check `docker compose -f docker-compose.prod.yml logs mysql`       |
| Google OAuth not working         | Google requires HTTPS for non-localhost redirects; you may need a domain first        |
| Out of disk space                | Run `docker system prune -af` and check `/var/lib/docker`                             |
| Containers keep restarting       | Check logs: `docker compose -f docker-compose.prod.yml logs <service>`                |
| Port 80 already in use           | Stop any existing web server: `systemctl stop nginx` or `systemctl stop apache2`      |