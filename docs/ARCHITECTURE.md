# Architecture & Strategy: HMSMaster Suite

## 1. Overview
HMSMaster is a specialized Management System designed to handle the complexity of hotel operations. It is part of a unified "Suite of Apps" alongside POSMaster.

## 2. Suite Strategy: HMS vs POSMaster
- **HMSMaster (Operation Layer):**
  - Focus: Property management, room inventory, reservation lifecycle, and "folio" accumulation.
  - Stakeholders: Receptionists, Housekeeping, House Managers.
  - Primary Goal: Ensuring every guest has a place to stay and every room is accounted for.
- **POSMaster (Transaction Layer):**
  - Focus: High-frequency sales, retail, inventory, and tax-compliant invoicing (facturas).
  - Stakeholders: Servers, Retail Staff, Accountants.
  - Primary Goal: Converting a reservation's needs into financial records.

**The Handshake:**
When a guest requests a "Minibar" or "Service", HMSMaster creates a **Folio Charge**. This charge is then synced to POSMaster (via the `syncChargeToPOSMaster` worker) to be included in the final checkout invoice.

## 3. Technology Trends (2026)
Based on investigation, the following themes are integrated:
- **Cloud-Native Ready:** Containerized architecture for easy horizontal scaling of the booking engine.
- **Mobile-First:** Responsive UI designed for receptionists on tablets.
- **Dynamic Pricing:** Data models support variable base rates (dynamic pricing).
- **API-Centric:** POSMaster integration follows a RESTful pattern with configurable endpoints.
- **Unified Aesthetic:** Using a shared "Dark Indigo" color palette to ensure a seamless user transition between apps.

## 4. Integration Contract (HMS $\rightarrow$ POSMaster)
- **Endpoint:** `POST /api/facturas`
- **Model:** `facturaRequestInput`
- **Mapping:**
  - `total_pagado` $\leftrightarrow$ HMS Folio Amount
  - `observaciones` $\leftrightarrow$ HMS Reservation ID + Guest Name
  - `detalles` $\leftrightarrow$ HMS Description + Price
