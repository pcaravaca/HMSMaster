# Production plan — HMSMaster (proposal, not executed)

## Decision to make
Two viable paths. Recommendation: **Option A** for the first production release.

### Option A — Single Ubuntu host with Docker Compose (recommended)
- Target: the existing Ubuntu server used for POSMaster (`192.168.0.108`) or the Tailscale host (`100.89.167.67`), in a dedicated directory `/opt/HMSMaster`.
- Access: Tailscale-only, no public DNS/certificate required for the first release; expose via `tailscale serve` on a friendly hostname if desired.
- Ports on the host: 18091 (API), 13101 (UI), 15434 (DB) to avoid colliding with POSMaster's 8081/3000/5432.
- Copy: `tar --exclude=.git --exclude=node_modules --exclude=dist -cf - . | ssh peter@HOST 'tar -C /opt/HMSMaster -xf -'`; create `env.production` on the server from `env.production.example` with real values (never copy the local one).
- Start: `cd /opt/HMSMaster && docker compose build --no-cache && docker compose up -d`.
- Verify: `curl localhost:18091/health`, `curl localhost:13101/api/stats`, `curl -I localhost:13101`.
- Backup: `docker exec hmsmaster-db-1 pg_dump -U hms_user hmsmaster > /backup/hmsmaster_$(date +%F).sql` on a daily cron; the Compose named volume `hmsmaster_hms_db_data` must be included in the host backup.
- Rollback: keep the previous image tag; `docker compose down` + restore the dump into a fresh volume; no schema migrations are destructive today (GORM `AutoMigrate` only adds).

### Option B — Public host with TLS
- Same Compose stack behind the existing nginx/Caddy reverse proxy, with a DNS name and Let's Encrypt certificate.
- Adds an auth gate: enable the JWT config described below before any public exposure.

## Blockers before going live
1. ~~**Authentication** — the API is open.~~ **DONE (v0.3.0):** optional JWT auth. Set `HMS_JWT_SECRET`, `HMS_ADMIN_USER` and `HMS_ADMIN_PASSWORD` in `env.production` to require `POST /api/auth/login` + Bearer token on all `/api` routes; leaving the secret empty keeps local development open. The frontend shows a login overlay on 401. Verified: no token 401, valid token 200, tampered token 401, bad credentials 401 (Go tests + live container).
2. **POSMaster bridge credentials** — `POSMASTER_API_URL`, `POSMASTER_API_TOKEN` (admin/cajero JWT), `POSMASTER_TIENDA_ID`, plus `POSMASTER_FORMA_PAGO` and CR IVA rate confirmation (currently hardcoded 13%).
3. **Production seed** — replace `seed.go` demo hotel/rooms with real property data or make seeding opt-in via env var.
4. **Backup retention** — define where dumps live and how many days are kept.

## Verification checklist (run after any deploy)
- `GET /health` → `{"status":"healthy","version":"0.3.0",...}`
- `GET /api/stats` shows real counts, not demo numbers.
- UI loads at the chosen port; action buttons (Confirmar / Check-in / Check-out) work.
- Create a test folio charge: with the bridge configured, verify `sync_status=synced` and the POSMaster invoice id; without it, verify `local`.
- `docker compose ps` shows all three containers healthy.

This document is a proposal only. Deploy requires Peter's explicit authorization.
