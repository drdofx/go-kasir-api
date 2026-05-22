# POS System Upgrade Design

## Context

Current state: Go backend API (`go-kasir-api`) with clean layered architecture (Handler → Service → Repository → DB). Features: Products CRUD, Categories CRUD, Transactional checkout, basic Sales Reports, JWT auth. No frontend (separate Next.js project at `next-kasir-web`). Frontend expects `GET /api/transactions` and `POST /api/auth/change-password` which don't exist yet.

Goal: Upgrade to full-featured POS backend with API versioning, domain modularity, and comprehensive feature set.

## Project Structure

```
internal/
├── domain/
│   ├── auth/                  Login, JWT, change password
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── product/               Products CRUD + search
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── category/              Categories CRUD
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── transaction/           Checkout, returns, refunds
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── customer/              Pelanggan + purchase history
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── inventory/             Stock alerts, purchase orders, suppliers
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   └── report/                Dashboard analytics, reports
│       ├── handler.go
│       └── service.go
├── pkg/
│   ├── database/
│   ├── middleware/
│   ├── helpers/
│   └── router/
└── testutil/
```

## API Design

All new endpoints under `/api/v1/*`. Existing `/api/*` endpoints kept for backward compatibility during migration.

### Auth
```
POST   /api/v1/auth/login             Login
GET    /api/v1/auth/me                Current user
POST   /api/v1/auth/logout            Logout
POST   /api/v1/auth/change-password   Change password
```

### Products
```
GET    /api/v1/products               List + search by name
POST   /api/v1/products               Create
GET    /api/v1/products/{id}          Get by ID
PUT    /api/v1/products/{id}          Update
DELETE /api/v1/products/{id}          Delete
```

### Categories
```
GET    /api/v1/categories             List + search
POST   /api/v1/categories             Create
GET    /api/v1/categories/{id}        Get by ID
PUT    /api/v1/categories/{id}        Update
DELETE /api/v1/categories/{id}        Delete
```

### Transactions
```
GET    /api/v1/transactions           List + search by date/product
GET    /api/v1/transactions/{id}      Detail with items
POST   /api/v1/checkout               Process checkout (with payment support)
```

### Customers
```
GET    /api/v1/customers              List + search
POST   /api/v1/customers              Create
GET    /api/v1/customers/{id}         Detail + purchase history
PUT    /api/v1/customers/{id}         Update
DELETE /api/v1/customers/{id}         Delete
```

### Returns & Refunds
```
POST   /api/v1/returns                Return items from a transaction
GET    /api/v1/returns                List returns
GET    /api/v1/returns/{id}           Detail return
```

### Payments
```
GET    /api/v1/payment-types          List payment types
```

### Suppliers & Purchase Orders
```
GET    /api/v1/suppliers              Supplier CRUD
POST   /api/v1/suppliers
PUT    /api/v1/suppliers/{id}
DELETE /api/v1/suppliers/{id}

GET    /api/v1/purchase-orders        PO CRUD
POST   /api/v1/purchase-orders
PUT    /api/v1/purchase-orders/{id}/receive  Receive PO (restock)
```

### Inventory
```
GET    /api/v1/inventory/alerts           Low stock alerts
PUT    /api/v1/inventory/alerts/{product_id} — Set min/max stock
```

### Reports
```
GET    /api/v1/report/dashboard       Dashboard analytics
GET    /api/v1/report/hari-ini        Today's summary
GET    /api/v1/report                 Date range report
GET    /api/v1/report/daily           Daily report
GET    /api/v1/report/weekly          Weekly summary
GET    /api/v1/report/monthly         Monthly summary
GET    /api/v1/report/by-category     Sales by category
GET    /api/v1/report/by-product      Sales by product
GET    /api/v1/report/export          Export CSV
```

### Receipts
```
GET    /api/v1/receipts/{transaction_id}  Get receipt data
```

### Users / RBAC
```
GET    /api/v1/users                  List users
POST   /api/v1/users                  Create user
PUT    /api/v1/users/{id}/role        Update role/permissions
```

## Database Schema

### New Tables

```sql
-- customers
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    email VARCHAR(255),
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- suppliers
CREATE TABLE suppliers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    contact_person VARCHAR(255),
    phone VARCHAR(50),
    email VARCHAR(255),
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- purchase_orders
CREATE TABLE purchase_orders (
    id SERIAL PRIMARY KEY,
    supplier_id INTEGER REFERENCES suppliers(id),
    status VARCHAR(20) DEFAULT 'pending',
    total_amount INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    received_at TIMESTAMP
);

CREATE TABLE purchase_order_items (
    id SERIAL PRIMARY KEY,
    purchase_order_id INTEGER REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
    subtotal INTEGER NOT NULL CHECK (subtotal >= 0)
);

-- payment_types + transaction_payments
CREATE TABLE payment_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE transaction_payments (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER REFERENCES transactions(id) ON DELETE CASCADE,
    payment_type_id INTEGER REFERENCES payment_types(id),
    amount INTEGER NOT NULL CHECK (amount > 0)
);

-- returns
CREATE TABLE returns (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER REFERENCES transactions(id),
    total_refund INTEGER NOT NULL CHECK (total_refund >= 0),
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE return_items (
    id SERIAL PRIMARY KEY,
    return_id INTEGER REFERENCES returns(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    subtotal INTEGER NOT NULL CHECK (subtotal >= 0)
);

-- inventory_alerts
CREATE TABLE inventory_alerts (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id) UNIQUE,
    min_stock INTEGER NOT NULL DEFAULT 0,
    max_stock INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN DEFAULT true
);

-- receipts
CREATE TABLE receipts (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER REFERENCES transactions(id) UNIQUE,
    receipt_number VARCHAR(50) NOT NULL UNIQUE,
    printed_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Existing Table Changes
- `users`: add `permissions TEXT[]` column for granular RBAC
- `transactions`: add `customer_id`, `payment_status` columns

## Checkout Flow (Updated)

```
POST /api/v1/checkout
{
  "items": [{"product_id": 1, "quantity": 2}],
  "customer_id": 1,               // optional
  "payments": [
    {"type": "cash", "amount": 50000},
    {"type": "qris", "amount": 25000}
  ]
}
```

Support split payment. Auto-deduct stock. Auto-generate receipt number. Auto-check inventory alerts.

## Implementation Order

| # | Phase | Description |
|---|-------|-------------|
| 1 | Refactor | Restructure to domain packages + router `/api/v1/*` + dual-run old routes |
| 2 | Gap Fix | `GET /api/transactions`, `POST /api/auth/change-password` |
| 3 | Customers | CRUD customers + link to transactions |
| 4 | Payments | Payment types + split payment in checkout |
| 5 | Returns | Return/refund items with stock restoration |
| 6 | Suppliers & PO | Supplier CRUD, purchase orders, restock |
| 7 | Inventory Alerts | Min/max stock alerts |
| 8 | Reports & Dashboard | Dashboard analytics, weekly/monthly, export CSV |
| 9 | RBAC | Granular permissions, user management |
| 10 | Receipts | Auto-generate receipt number, receipt endpoint |

Each phase includes tests (TDD). Phases 1-2 are highest priority (frontend gap).

## Future Plans (Post-Phase 10)

- Multi-organization / multi-branch
- Loyalty program
- E-commerce sync
- Offline mode
- AI demand forecasting
- Mobile app
