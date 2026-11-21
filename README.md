# HMSMaster (Hotel Management System Master)

Unified Management Suite for Hotels & Resorts.
Companion system to 🛒 POSMaster.

## Features
- **Real-time Dashboard**: Occupancy, arrivals, and in-house stats.
- **Room Management**: Availability tracking, pricing, and types.
- **Reservation Engine**: Multi-channel booking (Airbnb, Direct, etc.).
- **POSMaster Integration**: Seamless forwarding of folio charges to POSMaster for official invoicing.
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

## Suite Strategy
HMSMaster handles the "Long Stay" & "Stay Experience" (Rooms, check-ins, multi-day folios), while POSMaster handles the "Point of Sale" (Retail, restaurants, instantaneous sales, and final payment settlement).
