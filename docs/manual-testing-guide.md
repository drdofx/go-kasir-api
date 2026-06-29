# Manual Smoke Testing Guide

This guide is a practical smoke test, not an exhaustive API test suite. Run it after local changes to verify the main production flows still work.

## 1. Start the Stack

For a clean local run:

```bash
docker compose down -v
cp .env.example .env
docker compose up --build -d
```

Load local environment values into your shell:

```bash
set -a
. ./.env
set +a

BASE="http://localhost:8080"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-kasir123}"
```

Health check:

```bash
curl -s "$BASE/health" | jq .
```

Expected: `{"status":"OK"}`.

## 2. Auth, Refresh, and Branch

Login once and capture both tokens:

```bash
LOGIN_JSON=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"kasir\",\"password\":\"$ADMIN_PASSWORD\"}")

TOKEN=$(echo "$LOGIN_JSON" | jq -r '.access_token')
REFRESH_TOKEN=$(echo "$LOGIN_JSON" | jq -r '.refresh_token')

echo "$LOGIN_JSON" | jq .
```

Verify current user:

```bash
curl -s "$BASE/api/v1/auth/me" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Refresh the access token:

```bash
TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq -r '.access_token')
```

List branches and switch to the default seeded branch:

```bash
curl -s "$BASE/api/v1/branches" \
  -H "Authorization: Bearer $TOKEN" | jq .

TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/switch-branch" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"branch_id":1}' | jq -r '.token')
```

## 3. Pagination Smoke Test

List endpoints support `page` and `per_page`. Headers include `X-Page`, `X-Per-Page`, and `X-Total-Count`.

```bash
curl -i -s "$BASE/api/v1/products?page=1&per_page=10" \
  -H "Authorization: Bearer $TOKEN"
```

Expected: `200 OK` and pagination headers.

Invalid pagination should fail:

```bash
curl -i -s "$BASE/api/v1/products?page=0" \
  -H "Authorization: Bearer $TOKEN"
```

Expected: `400 Bad Request`.

## 4. Catalog Flow

Create category:

```bash
CATEGORY_JSON=$(curl -s -X POST "$BASE/api/v1/categories" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Beverages","description":"Drink products"}')

CATEGORY_ID=$(echo "$CATEGORY_JSON" | jq -r '.id')
echo "$CATEGORY_JSON" | jq .
```

Create product:

```bash
PRODUCT_JSON=$(curl -s -X POST "$BASE/api/v1/products" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Black Coffee\",\"price\":12000,\"category_id\":$CATEGORY_ID}")

PRODUCT_ID=$(echo "$PRODUCT_JSON" | jq -r '.id')
echo "$PRODUCT_JSON" | jq .
```

Set branch stock directly for local testing:

```bash
docker compose exec -T db psql -U kasir -d kasir \
  -c "INSERT INTO product_stocks (product_id, branch_id, stock) VALUES ($PRODUCT_ID, 1, 20) ON CONFLICT (product_id, branch_id) DO UPDATE SET stock = EXCLUDED.stock;"
```

Check products:

```bash
curl -s "$BASE/api/v1/products?name=coffee" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

## 5. Customer Flow

Create customer:

```bash
CUSTOMER_JSON=$(curl -s -X POST "$BASE/api/v1/customers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"John Customer","phone":"08123456789","email":"john@example.com","address":"Customer Street"}')

CUSTOMER_ID=$(echo "$CUSTOMER_JSON" | jq -r '.id')
echo "$CUSTOMER_JSON" | jq .
```

Search customers:

```bash
curl -s "$BASE/api/v1/customers?search=john&page=1&per_page=10" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

## 6. Supplier and Purchase Order Flow

Create supplier:

```bash
SUPPLIER_JSON=$(curl -s -X POST "$BASE/api/v1/suppliers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Supply","contact_person":"Alice","phone":"0800000000","email":"alice@example.com","address":"Warehouse Street"}')

SUPPLIER_ID=$(echo "$SUPPLIER_JSON" | jq -r '.id')
echo "$SUPPLIER_JSON" | jq .
```

Create purchase order:

```bash
PO_JSON=$(curl -s -X POST "$BASE/api/v1/purchase-orders" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"supplier_id\":$SUPPLIER_ID,\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":5,\"unit_price\":9000}]}")

PO_ID=$(echo "$PO_JSON" | jq -r '.id')
echo "$PO_JSON" | jq .
```

Receive purchase order:

```bash
curl -s -X POST "$BASE/api/v1/purchase-orders/$PO_ID/receive" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Expected: status becomes `received`, and branch stock increases.

## 7. Checkout Flow

Create checkout with explicit payment:

```bash
TX_JSON=$(curl -s -X POST "$BASE/api/v1/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"customer_id\":$CUSTOMER_ID,\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":2}],\"payments\":[{\"type\":\"cash\",\"amount\":24000}]}")

TX_ID=$(echo "$TX_JSON" | jq -r '.id')
echo "$TX_JSON" | jq .
```

List transactions:

```bash
curl -s "$BASE/api/v1/transactions?page=1&per_page=10" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Customer purchase history:

```bash
curl -s "$BASE/api/v1/customers/$CUSTOMER_ID/purchases" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

## 8. Returns Flow

Return one item from the checkout:

```bash
RETURN_JSON=$(curl -s -X POST "$BASE/api/v1/returns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"transaction_id\":$TX_ID,\"reason\":\"Customer changed item\",\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":1}]}")

RETURN_ID=$(echo "$RETURN_JSON" | jq -r '.id')
echo "$RETURN_JSON" | jq .
```

Get return:

```bash
curl -s "$BASE/api/v1/returns/$RETURN_ID" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

## 9. Inventory Alerts

Set threshold:

```bash
curl -s -X PUT "$BASE/api/v1/inventory/alerts/$PRODUCT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"min_stock":50,"max_stock":100,"enabled":true}' | jq .
```

List alerts:

```bash
curl -s "$BASE/api/v1/inventory/alerts" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Expected: the product appears if branch stock is at or below `min_stock`.

## 10. Reports

Dashboard:

```bash
curl -s "$BASE/api/v1/report/dashboard" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Today:

```bash
curl -s "$BASE/api/v1/report/today" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Date range:

```bash
TODAY=$(date +%F)
curl -s "$BASE/api/v1/report?start_date=$TODAY&end_date=$TODAY" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

CSV export:

```bash
curl -s "$BASE/api/v1/report/export?start_date=$TODAY&end_date=$TODAY" \
  -H "Authorization: Bearer $TOKEN"
```

## 11. Refresh Token Revocation

Logout and revoke the refresh token:

```bash
curl -s -X POST "$BASE/api/v1/auth/logout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq .
```

Try refreshing again:

```bash
curl -i -s -X POST "$BASE/api/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
```

Expected: `401 Unauthorized`.

## 12. Production Config Smoke Test

The API should fail fast when production secrets are weak:

```bash
APP_ENV=production JWT_SECRET=short go run ./cmd/api
```

Expected: startup fails with a `JWT_SECRET` validation error.

## Notes

- Use `docker compose logs -f api` when a request returns `500`.
- Use `docker compose exec -T db psql -U kasir -d kasir` for database inspection.
- This guide intentionally uses seeded organization and branch `1`.
- For automated coverage, use `go test ./...`; this guide is only a manual confidence pass.
