# Manual API Testing Guide

This guide assumes the API is running locally on `http://localhost:8080`.

## Prerequisites

```bash
cp .env.example .env
docker compose up --build -d

BASE="http://localhost:8080"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-kasir123}"
```

The development seed creates a `kasir` admin user when the `users` table is empty.

## Authentication

Login:

```bash
TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"kasir\",\"password\":\"$ADMIN_PASSWORD\"}" | jq -r '.token')

REFRESH_TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"kasir\",\"password\":\"$ADMIN_PASSWORD\"}" | jq -r '.refresh_token')
```

Get current user:

```bash
curl -s "$BASE/api/v1/auth/me" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Change password:

```bash
curl -s -X POST "$BASE/api/v1/auth/change-password" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"kasir123","new_password":"new-secure-password"}' | jq .
```

Switch active branch:

```bash
curl -s -X POST "$BASE/api/v1/auth/switch-branch" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"branch_id":1}' | jq .
```

Use the returned token for branch-scoped operations.

Refresh access token:

```bash
TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq -r '.access_token')
```

Logout and revoke refresh token:

```bash
curl -s -X POST "$BASE/api/v1/auth/logout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq .
```

## Admin Operations

List users:

```bash
curl -s "$BASE/api/v1/users?page=1&per_page=20" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Create user in the current organization:

```bash
curl -s -X POST "$BASE/api/v1/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"cashier1","password":"cashier-password","name":"Cashier One","role":"cashier"}' | jq .
```

Update user role and permissions:

```bash
curl -s -X PUT "$BASE/api/v1/users/2" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role":"admin","permissions":["product:create","product:delete"]}' | jq .
```

List branches:

```bash
curl -s "$BASE/api/v1/branches" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Create branch:

```bash
curl -s -X POST "$BASE/api/v1/branches" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Second Branch","code":"SECOND","address":"Main Street 2"}' | jq .
```

Update branch:

```bash
curl -s -X PUT "$BASE/api/v1/branches/2" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Second Branch Updated","code":"SECOND","address":"Main Street 5"}' | jq .
```

## Catalog

Create category:

```bash
curl -s -X POST "$BASE/api/v1/categories" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Beverages","description":"Drink products"}' | jq .
```

Create product:

```bash
curl -s -X POST "$BASE/api/v1/products" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Black Coffee","price":10000,"category_id":1}' | jq .
```

List products:

```bash
curl -s "$BASE/api/v1/products?name=coffee" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Get, update, and delete product:

```bash
curl -s "$BASE/api/v1/products/1" -H "Authorization: Bearer $TOKEN" | jq .

curl -s -X PUT "$BASE/api/v1/products/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Black Coffee Large","price":12000,"category_id":1}' | jq .

curl -i -X DELETE "$BASE/api/v1/products/1" \
  -H "Authorization: Bearer $TOKEN"
```

## Customers

Create customer:

```bash
curl -s -X POST "$BASE/api/v1/customers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"John Customer","phone":"08123456789","email":"john@example.com","address":"Customer Street"}' | jq .
```

List and search customers:

```bash
curl -s "$BASE/api/v1/customers?search=john" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Get purchase history:

```bash
curl -s "$BASE/api/v1/customers/1/purchases" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

## Checkout

Checkout requires branch stock rows in `product_stocks`. After seeding a product, create or update stock directly during manual local testing if needed:

```bash
docker compose exec db psql -U kasir -d kasir \
  -c "INSERT INTO product_stocks (product_id, branch_id, stock) VALUES (1, 1, 20) ON CONFLICT (product_id, branch_id) DO UPDATE SET stock = EXCLUDED.stock;"
```

Create a transaction with explicit payment:

```bash
curl -s -X POST "$BASE/api/v1/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"branch_id":1,"items":[{"product_id":1,"quantity":2}],"payments":[{"type":"cash","amount":20000}]}' | jq .
```

List transactions:

```bash
curl -s "$BASE/api/v1/transactions" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

## Reports

```bash
curl -s "$BASE/api/v1/report/dashboard" \
  -H "Authorization: Bearer $TOKEN" | jq .

curl -s "$BASE/api/v1/report/today" \
  -H "Authorization: Bearer $TOKEN" | jq .

curl -s "$BASE/api/v1/report?start_date=2026-01-01&end_date=2026-01-31" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

## Production Configuration Smoke Test

The API should fail fast when production secrets are missing or weak:

```bash
APP_ENV=production JWT_SECRET=short go run ./cmd/api
```

Use a random `JWT_SECRET` with at least 32 characters in real production environments.
