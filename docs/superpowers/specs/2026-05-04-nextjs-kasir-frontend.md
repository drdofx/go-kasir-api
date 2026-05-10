# Next.js Kasir Frontend — Design Spec

> **Project:** `next-kasir-web` at `/Users/dofx/projects/personal/golang-course/next-kasir-web`
> **Backend:** `go-kasir-api` at `/Users/dofx/projects/personal/golang-course/product`

---

## 1. Overview

A Next.js 14 cashier (POS) web application for the `go-kasir-api` backend. Handles product management, category management, checkout transactions, and sales reporting with a modern dashboard UI.

## 2. Tech Stack

| Layer | Technology |
|---|---|
| Framework | Next.js 14 (App Router) |
| Language | TypeScript 5 (strict mode) |
| Styling | Tailwind CSS 3.4 |
| UI Components | shadcn/ui |
| Data Fetching | TanStack Query v5 |
| Client State | Zustand |
| Forms | react-hook-form + zod |
| Icons | lucide-react |
| Testing | Vitest + @testing-library/react |

## 3. Architecture

### 3.1 Directory Structure

```
next-kasir-web/
├── src/
│   ├── app/                          # Next.js App Router
│   │   ├── auth/
│   │   │   └── login/
│   │   │       └── page.tsx          # Login page
│   │   ├── dashboard/
│   │   │   ├── layout.tsx            # Dashboard shell (sidebar + header)
│   │   │   ├── page.tsx              # Dashboard home
│   │   │   ├── products/
│   │   │   │   ├── page.tsx          # Product list
│   │   │   │   └── new/
│   │   │   │       └── page.tsx      # Create product
│   │   │   ├── categories/
│   │   │   │   └── page.tsx          # Category CRUD
│   │   │   ├── reports/
│   │   │   │   └── page.tsx          # Sales reports
│   │   │   ├── transactions/
│   │   │   │   └── page.tsx          # Transaction history
│   │   │   └── settings/
│   │   │       └── page.tsx          # Settings
│   │   ├── pos/
│   │   │   └── page.tsx              # POS cashier (full-screen)
│   │   ├── layout.tsx                # Root layout (providers)
│   │   ├── page.tsx                  # Redirect to /dashboard
│   │   └── error.tsx                 # Global error boundary
│   ├── components/
│   │   ├── ui/                       # shadcn/ui components
│   │   ├── layout/                   # Sidebar, Header, DashboardMain
│   │   ├── pos/                      # ProductGrid, Cart, CheckoutModal
│   │   ├── products/                 # ProductTable, ProductForm
│   │   ├── categories/               # CategoryTable, CategoryForm
│   │   └── common/                   # DataTable, PageHeader, LoadingSpinner
│   ├── hooks/
│   │   ├── api/                      # TanStack Query hooks
│   │   ├── useAuth.ts                # Auth state + login/logout
│   │   └── useCart.ts                # POS cart state
│   ├── lib/
│   │   ├── api-client.ts             # Fetch wrapper with auth
│   │   ├── query-client.ts           # TanStack Query config
│   │   └── utils.ts                  # cn(), helpers
│   ├── stores/
│   │   ├── auth.store.ts             # Zustand: user, token, isAuthenticated
│   │   └── cart.store.ts             # Zustand: cart items, total
│   ├── types/
│   │   └── api.ts                    # TypeScript types
│   └── validations/
│       └── schemas.ts                # Zod schemas
├── public/
├── components.json                   # shadcn/ui config
├── tailwind.config.ts
├── next.config.mjs
├── tsconfig.json
└── package.json
```

### 3.2 Auth Pattern

- JWT token stored in `localStorage`
- `api-client.ts` reads token, attaches `Authorization: Bearer <token>`
- `useAuth` hook: login (store token + user), logout (clear storage), restore on mount
- Dashboard layout: redirect to `/auth/login` if not authenticated
- POS page: also protected

### 3.3 API Client

```typescript
// src/lib/api-client.ts
const apiClient = {
  get: (url, config?) => fetch(BASE_URL + url, { ...config, headers: { Authorization: `Bearer ${getToken()}` } }),
  post: (url, data, config?) => fetch(BASE_URL + url, { method: 'POST', body: JSON.stringify(data), ... }),
  // ... put, delete
}
```

### 3.4 TanStack Query Patterns

| Resource | Query Key | Stale Time |
|---|---|---|
| Products | `['products']` | 5 min |
| Product detail | `['product', id]` | 5 min |
| Categories | `['categories']` | 10 min |
| Reports | `['report', date]` | 0 (always fresh) |
| Transactions | `['transactions']` | 1 min |

Mutations invalidate related queries on success.

## 4. Pages

### 4.1 Auth

**Login** (`/auth/login`)
- Username/password form (react-hook-form + zod)
- On success: store token in localStorage, set auth state, redirect to `/dashboard`
- Public route (no sidebar)

### 4.2 Dashboard

**Dashboard Home** (`/dashboard`)
- Protected route with sidebar layout
- Today's sales summary card (revenue, transaction count, top product)
- Quick stats: total products, total categories
- Recent transactions list (last 5)

### 4.3 POS (Cashier)

**POS Screen** (`/pos`)
- Full-screen layout (no sidebar, minimal chrome)
- Left: Product grid with search bar + category filter tabs
- Right: Cart sidebar
  - List of added items (product name, qty, price, subtotal)
  - Quantity +/- buttons
  - Remove item button
  - Total amount
  - Checkout button
- Checkout flow:
  1. Click Checkout → confirmation modal
  2. Call `POST /api/checkout`
  3. On success: show receipt modal, clear cart
  4. Print receipt (optional)

### 4.4 Products

**Product List** (`/dashboard/products`)
- DataTable with columns: ID, Name, Price, Stock, Category, Actions
- Search by name (debounced)
- Pagination
- Actions: Edit (navigate), Delete (confirm modal)
- "Add Product" button → `/dashboard/products/new`

**Create/Edit Product** (`/dashboard/products/new` or modal)
- Form fields: Name, Price, Stock, Category (dropdown)
- Zod validation
- On success: invalidate products query, navigate back

### 4.5 Categories

**Category List** (`/dashboard/categories`)
- DataTable: ID, Name, Description, Actions
- Inline create/edit (or modal)
- Delete with confirmation

### 4.6 Reports

**Reports** (`/dashboard/reports`)
- Today's report card (auto-load on mount)
- Date range picker (start_date, end_date)
- "Generate Report" button
- Display: total revenue, total transactions, top product

### 4.7 Transactions

**Transaction History** (`/dashboard/transactions`)
- DataTable: ID, Total Amount, Created At, Item Count
- Click to view details (modal or expand row)
- No pagination needed for MVP (or simple offset pagination)

### 4.8 Settings

**Settings** (`/dashboard/settings`)
- Change password form
- App info

## 5. State Management

### Zustand Stores

**Auth Store** (`src/stores/auth.store.ts`)
```typescript
interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  setAuth: (token: string, user: User) => void
  logout: () => void
}
```

**Cart Store** (`src/stores/cart.store.ts`)
```typescript
interface CartState {
  items: CartItem[]
  addItem: (product: Product) => void
  removeItem: (productId: number) => void
  updateQty: (productId: number, qty: number) => void
  clear: () => void
  total: number
}
```

## 6. Components

### Reusable Components

**DataTable**
- Built on shadcn/ui Table
- Props: columns, data, loading, pagination
- Used by: Products, Categories, Transactions

**PageHeader**
- Title + breadcrumb + action button
- Used by all dashboard pages

**Form Components**
- FormInput, FormSelect, FormNumber (wrap shadcn inputs with react-hook-form)

### POS Components

**ProductGrid**
- Grid of product cards (image placeholder, name, price, stock)
- Filter by category tabs
- Search input (debounced)
- Click to add to cart

**CartSidebar**
- Scrollable list of cart items
- Quantity controls
- Total calculation
- Checkout button

**CheckoutModal**
- Confirm order summary
- Call API on confirm
- Show success/receipt

## 7. API Integration

### Backend Base URL

```
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| POST | `/api/auth/login` | Login |
| GET | `/api/auth/me` | Get current user |
| GET | `/api/products` | List products |
| POST | `/api/products` | Create product |
| GET | `/api/products/{id}` | Get product |
| PUT | `/api/products/{id}` | Update product |
| DELETE | `/api/products/{id}` | Delete product |
| GET | `/api/categories` | List categories |
| POST | `/api/categories` | Create category |
| PUT | `/api/categories/{id}` | Update category |
| DELETE | `/api/categories/{id}` | Delete category |
| POST | `/api/checkout` | Create transaction |
| GET | `/api/report/hari-ini` | Today's report |
| GET | `/api/report?start=&end=` | Date range report |

## 8. Error Handling

- **401**: Clear auth state, redirect to login
- **403**: Show forbidden page
- **500**: Toast error message
- Network errors: Retry once, then toast

## 9. Styling

- Tailwind CSS with custom color variables
- shadcn/ui default theme (customize as needed)
- Responsive: Dashboard works on tablet; POS optimized for desktop/tablet
- Dark mode: optional (shadcn supports it)

## 10. Testing

- Unit tests for stores (Zustand)
- Unit tests for hooks (TanStack Query)
- Component tests for forms (react-testing-library)
- Coverage threshold: 70%
