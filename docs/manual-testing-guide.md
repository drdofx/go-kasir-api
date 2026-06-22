# Manual API Testing Guide

## Prerequisites

```bash
# Start the server
docker-compose up -d        # or: go run ./cmd/api/

# Set base URL
BASE="http://localhost:8080"
```

---

## 1. Auth

### Login (get JWT token)
```bash
TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"kasir","password":"kasir123"}' | jq -r '.token')
echo "$TOKEN"
```

### Get current user
```bash
curl -s "$BASE/api/v1/auth/me" -H "Authorization: Bearer $TOKEN" | jq .
```

### Change password
```bash
curl -s -X POST "$BASE/api/v1/auth/change-password" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"kasir123","new_password":"kasir456"}' | jq .
```

### Switch branch
```bash
curl -s -X POST "$BASE/api/v1/auth/switch-branch" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"branch_id":1}' | jq .
```

### Logout
```bash
curl -s -X POST "$BASE/api/v1/auth/logout" -H "Authorization: Bearer $TOKEN" | jq .
```

---

## 2. Users (Admin)

### List users
```bash
curl -s "$BASE/api/v1/users" -H "Authorization: Bearer $TOKEN" | jq .
```

### Create user
```bash
curl -s -X POST "$BASE/api/v1/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"kasir2","password":"kasir123","name":"Kasir Dua","role":"cashier"}' | jq .
```

### Update user role
```bash
curl -s -X PUT "$BASE/api/v1/users/2" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role":"admin","permissions":["product:create","product:delete"]}' | jq .
```

---

## 3. Branches

### List branches (org-scoped)
```bash
curl -s "$BASE/api/v1/branches" -H "Authorization: Bearer $TOKEN" | jq .
```

### Create branch
```bash
curl -s -X POST "$BASE/api/v1/branches" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Cabang B","code":"B","address":"Jl. Merdeka No.2"}' | jq .
```

### Update branch
```bash
curl -s -X PUT "$BASE/api/v1/branches/2" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Cabang B Update","code":"B2","address":"Jl. Baru No.5"}' | jq .
```

---

## 4. Categories

### List categories
```bash
curl -s "$BASE/api/v1/categories" -H "Authorization: Bearer $TOKEN" | jq .
```

### Create category
```bash
curl -s -X POST "$BASE/api/v1/categories" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Minuman","description":"Minuman Segar"}' | jq .
```

### Get category by ID
```bash
curl -s "$BASE/api/v1/categories/1" -H "Authorization: Bearer $TOKEN" | jq .
```

### Update category
```bash
curl -s -X PUT "$BASE/api/v1/categories/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Minuman Dingin","description":"Minuman Dingin Segar"}' | jq .
```

### Delete category
```bash
curl -s -X DELETE "$BASE/api/v1/categories/1" -H "Authorization: Bearer $TOKEN"
```

---

## 5. Products

### List products
```bash
curl -s "$BASE/api/v1/products" -H "Authorization: Bearer $TOKEN" | jq .
```

### Search products by name
```bash
curl -s "$BASE/api/v1/products?name=kopi" -H "Authorization: Bearer $TOKEN" | jq .
```

### Create product
```bash
curl -s -X POST "$BASE/api/v1/products" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Kopi Hitam","price":10000,"category_id":1}' | jq .
```

### Get product by ID (shows per-branch stocks)
```bash
curl -s "$BASE/api/v1/products/1" -H "Authorization: Bearer $TOKEN" | jq .
```

### Update product
```bash
curl -s -X PUT "$BASE/api/v1/products/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Kopi Hitam Update","price":12000,"category_id":1}' | jq .
```

### Delete product
```bash
curl -s -X DELETE "$BASE/api/v1/products/1" -H "Authorization: Bearer $TOKEN"
```

---

## 6. Customers

### List customers (with search)
```bash
curl -s "$BASE/api/v1/customers" -H "Authorization: Bearer $TOKEN" | jq .
curl -s "$BASE/api/v1/customers?search=budi" -H "Authorization: Bearer $TOKEN" | jq .
```

### Create customer
```bash
curl -s -X POST "$BASE/api/v1/customers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Budi Santoso","phone":"08123456789","email":"budi@mail.com"}' | jq .
```

### Get customer by ID
```bash
curl -s "$BASE/api/v1/customers/1" -H "Authorization: Bearer $TOKEN" | jq .
```

### Get customer purchase history
```bash
curl -s "$BASE/api/v1/customers/1/history" -H "Authorization: Bearer $TOKEN" | jq .
```

### Update customer
```bash
curl -s -X PUT "$BASE/api/v1/customers/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Budi Update","phone":"08123456789","email":"budi@mail.com","address":"Jl. Baru"}' | jq .
```

### Delete customer
```bash
curl -s -X DELETE "$BASE/api/v1/customers/1" -H "Authorization: Bearer $TOKEN"
```

---

## 7. Payment Types

### List payment types
```bash
curl -s "$BASE/api/v1/payment-types" -H "Authorization: Bearer $TOKEN" | jq .
```

---

## 8. Transactions (Checkout)

**Before checkout**, make sure:
1. You have at least 1 product with stock in your branch
2. You're logged in (token set)
3. You've switched to a branch (`branch_id` in JWT)

### Checkout — simple (cash only)
```bash
curl -s -X POST "$BASE/api/v1/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":1,"quantity":2}]}' | jq .
```

### Checkout — with customer + split payment
```bash
curl -s -X POST "$BASE/api/v1/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items":[{"product_id":1,"quantity":2},{"product_id":2,"quantity":1}],
    "customer_id":1,
    "payments":[
      {"type":"cash","amount":15000},
      {"type":"qris","amount":15000}
    ]
  }' | jq .
```

### List transactions
```bash
curl -s "$BASE/api/v1/transactions" -H "Authorization: Bearer $TOKEN" | jq .
```

### Get transaction detail
```bash
curl -s "$BASE/api/v1/transactions/1" -H "Authorization: Bearer $TOKEN" | jq .
```

---

## 9. Returns

### Return items from a transaction
```bash
curl -s -X POST "$BASE/api/v1/returns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"transaction_id":1,"reason":"Rusak","items":[{"product_id":1,"quantity":1}]}' | jq .
```

### List returns
```bash
curl -s "$BASE/api/v1/returns" -H "Authorization: Bearer $TOKEN" | jq .
```

### Get return detail
```bash
curl -s "$BASE/api/v1/returns/1" -H "Authorization: Bearer $TOKEN" | jq .
```

---

## 10. Suppliers

### List suppliers
```bash
curl -s "$BASE/api/v1/suppliers" -H "Authorization: Bearer $TOKEN" | jq .
```

### Create supplier
```bash
curl -s -X POST "$BASE/api/v1/suppliers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"PT Kopi Makmur","contact_person":"Andi","phone":"08111111111"}' | jq .
```

### Update supplier
```bash
curl -s -X PUT "$BASE/api/v1/suppliers/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"PT Kopi Makmur Update","contact_person":"Andi Baru"}' | jq .
```

### Delete supplier
```bash
curl -s -X DELETE "$BASE/api/v1/suppliers/1" -H "Authorization: Bearer $TOKEN"
```

---

## 11. Purchase Orders

### Create purchase order
```bash
curl -s -X POST "$BASE/api/v1/purchase-orders" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"supplier_id":1,"items":[{"product_id":1,"quantity":50,"unit_price":8000}]}' | jq .
```

### List purchase orders
```bash
curl -s "$BASE/api/v1/purchase-orders" -H "Authorization: Bearer $TOKEN" | jq .
```

### Get purchase order detail
```bash
curl -s "$BASE/api/v1/purchase-orders/1" -H "Authorization: Bearer $TOKEN" | jq .
```

### Receive purchase order (restock)
```bash
curl -s -X PUT "$BASE/api/v1/purchase-orders/1/receive" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## 12. Inventory Alerts

### Set stock threshold for a product
```bash
curl -s -X PUT "$BASE/api/v1/inventory/alerts/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"min_stock":10,"max_stock":100,"enabled":true}' | jq .
```

### Get low stock alerts
```bash
curl -s "$BASE/api/v1/inventory/alerts" -H "Authorization: Bearer $TOKEN" | jq .
```

---

## 13. Reports

### Dashboard (today's analytics)
```bash
curl -s "$BASE/api/v1/report/dashboard" -H "Authorization: Bearer $TOKEN" | jq .
```

### Today's summary
```bash
curl -s "$BASE/api/v1/report/hari-ini" -H "Authorization: Bearer $TOKEN" | jq .
```

### Weekly report
```bash
curl -s "$BASE/api/v1/report/weekly" -H "Authorization: Bearer $TOKEN" | jq .
```

### Monthly report
```bash
curl -s "$BASE/api/v1/report/monthly" -H "Authorization: Bearer $TOKEN" | jq .
```

### Date range report
```bash
curl -s "$BASE/api/v1/report?start_date=2026-05-01&end_date=2026-05-22" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### Sales by category
```bash
curl -s "$BASE/api/v1/report/by-category?start_date=2026-05-01&end_date=2026-05-22" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### Sales by product
```bash
curl -s "$BASE/api/v1/report/by-product?start_date=2026-05-01&end_date=2026-05-22" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### Export CSV
```bash
curl -s "$BASE/api/v1/report/export?start_date=2026-05-01&end_date=2026-05-22" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 14. Receipts

### Get receipt by transaction ID
```bash
curl -s "$BASE/api/v1/receipts/1" -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Quick Smoke Test (all-in-one)

```bash
BASE="http://localhost:8080"
TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"kasir","password":"kasir123"}' | jq -r '.token')

echo "=== Branches ===" && curl -s "$BASE/api/v1/branches" -H "Authorization: Bearer $TOKEN" | jq .
echo "=== Categories ===" && curl -s "$BASE/api/v1/categories" -H "Authorization: Bearer $TOKEN" | jq .
echo "=== Products ===" && curl -s "$BASE/api/v1/products" -H "Authorization: Bearer $TOKEN" | jq .
echo "=== Dashboard ===" && curl -s "$BASE/api/v1/report/dashboard" -H "Authorization: Bearer $TOKEN" | jq .
echo "=== Payment Types ===" && curl -s "$BASE/api/v1/payment-types" -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Test Flow (recommended order)

1. **Login** → get token
2. **Branches** → list, make sure at least 1 branch
3. **Switch branch** → set `branch_id` in JWT (required for checkout)
4. **Categories** → create 2-3 categories
5. **Products** → create 3-5 products (no stock needed, stock is per-branch)
6. **Set product stock** via inventory alerts or direct DB: `INSERT INTO product_stocks (product_id, branch_id, stock) VALUES (1, 1, 100);`
7. **Customers** → create 1-2 customers
8. **Checkout** → process a sale (with/without customer, with/without split payment)
9. **Transactions** → list & view detail (includes auto-generated receipt number)
10. **Receipts** → get receipt by transaction ID
11. **Returns** → return an item from the transaction (stock restored)
12. **Dashboard** → check today's analytics
13. **Reports** → try date range, weekly, monthly, by-category, by-product, CSV export
14. **Suppliers** → create supplier
15. **Purchase Orders** → create PO, then receive it (stock increases)
16. **Inventory** → set thresholds, check low stock alerts
