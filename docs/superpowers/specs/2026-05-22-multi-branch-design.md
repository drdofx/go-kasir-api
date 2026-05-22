# Multi-Branch + Multi-User Design

## Context
Current: 1 user, no branch concept. Goal: multi-user in 1 org with branches, stock per branch, future-proof for multi-tenant.

## Schema
- `organizations` — tenant
- `branches` — org children
- `product_stocks` — per-branch stock (replaces products.stock)
- All data tables: add `organization_id`, `branch_id` where relevant

## Auth
JWT extended with `org_id`, `branch_id`. `RequireOrg` middleware extracts org context. Switch-branch endpoint re-issues JWT.

## API
- `GET/POST /api/v1/branches` — CRUD
- `POST /api/v1/auth/switch-branch` — switch active branch
- Existing endpoints scoped by org+branch
- Product response includes stocks per branch

## Implementation Order
1. Migrations (orgs, branches, product_stocks)
2. Seed default org+branch on startup
3. Add org_id/branch_id to all tables, migrate data
4. JWT upgrade + RequireOrg middleware
5. Branch CRUD + switch endpoint
6. Product stock refactor
7. Transaction branch scope
