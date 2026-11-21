# HMSMaster (Hotel Management System Master)

Unified Management Suite for Hotels & Resorts.
Companion system to 🛒 POSMaster.

## Features
- **Real-time Dashboard**: Occupancy, arrivals, and in-house stats.
- **Room Management**: Availability tracking, pricing, and types.
- **Multi-Property**: switch the active property in the header; stats, reservations, rooms and availability are scoped to it (`?hotel_id=`). New properties can be added from the dashboard.
- **Reservation Engine**: Multi-channel booking (Airbnb, Direct, etc.) with overlap protection — a room cannot be sold twice for the same nights; maintenance rooms are also blocked.
- **Unified Guest Profiles**: `Guest` records with document, contact and stay history. Reservations link by `guest_id` and inherit name/email; `GET /api/guests/:id` returns the guest plus all their stays.
- **Availability Search**: `GET /api/availability?check_in=YYYY-MM-DD&check_out=YYYY-MM-DD` returns every room with its free/busy status and conflicting reservation ids.
- **POSMaster Integration**: Seamless forwarding of folio charges to POSMaster for official invoicing.
- **Resilient Invoicing**: each charge carries an idempotency key; outcomes are `synced`, `failed` or `uncertain` (unknown outcomes are never auto-retried). `POST /api/reservations/:id/charges/:chargeId/retry` looks for an existing invoice before re-posting, and a conservative worker retries only definitive failures up to 3 times. `GET /api/posmaster/status` exposes the counts.
- **Dark Indigo Theme**: Matches POSMaster's aura-dark-indigo visual language.

## Architecture
- **Backend**: Go (Gin + GORM)
- **Frontend**: Vue 3 + Vite
- **Database**: PostgreSQL
- **Infrastructure**: Docker Compose

## Local Development
1. `cp env.production.example env.production` and fill the real values.
2. `docker compose build --no-cache && docker compose up -d`
3. Backend: `http://localhost:8091`
4. Frontend: `http://localhost:3101`
5. PostgreSQL: `localhost:55434`

`env.production` is gitignored; never commit real credentials. The POSMaster
bridge is optional: leave `POSMASTER_API_URL` empty and folio charges stay local.
Demo data is opt-in (`HMS_SEED_DEMO=true`). Back up with `sh scripts/backup.sh`;
dumps land in `backups/` and older ones are pruned automatically.

## Suite Strategy
HMSMaster handles the "Long Stay" & "Stay Experience" (Rooms, check-ins, multi-day folios), while POSMaster handles the "Point of Sale" (Retail, restaurants, instantaneous sales, and final payment settlement).
