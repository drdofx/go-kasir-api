# Phase 3: Returns & Refunds Implementation Plan

**Goal:** Add return/refund functionality — return items from a transaction with stock restoration.

**Endpoints:**
- `POST /api/v1/returns` — Return items from a transaction
- `GET /api/v1/returns` — List all returns
- `GET /api/v1/returns/{id}` — Detail return

**Files:**
- `migrations/000008_create_returns_tables.up.sql`
- `migrations/000008_create_returns_tables.down.sql`
- `internal/domain/returns/repository.go`
- `internal/domain/returns/service.go`
- `internal/domain/returns/handler.go`
- `internal/domain/returns/returns_service_test.go`
- `cmd/api/main.go` (add routes)
