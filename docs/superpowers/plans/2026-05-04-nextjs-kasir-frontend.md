# Next.js Kasir Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-ready Next.js 14 POS cashier web app with dashboard, product management, checkout, and reporting.

**Architecture:** App Router with feature-based organization. Server Components by default, Client Components for interactivity. TanStack Query for server state, Zustand for client state. JWT auth with localStorage.

**Tech Stack:** Next.js 14, TypeScript 5, Tailwind CSS 3, shadcn/ui, TanStack Query v5, Zustand, react-hook-form + zod, lucide-react, Vitest

---

### Task 1: Project Initialization

**Files:**
- Create: `next-kasir-web/` (entire project)

- [ ] **Step 1: Create Next.js project with shadcn/ui**

Run:
```bash
cd /Users/dofx/projects/personal/golang-course
npx create-next-app@14 next-kasir-web --typescript --tailwind --eslint --app --src-dir --no-import-alias
```

- [ ] **Step 2: Initialize shadcn/ui**

```bash
cd next-kasir-web
npx shadcn-ui@latest init --yes --template next --base-color slate
```

- [ ] **Step 3: Install dependencies**

```bash
npm install @tanstack/react-query @tanstack/react-query-devtools zustand react-hook-form @hookform/resolvers zod axios lucide-react
npm install -D @testing-library/react @testing-library/jest-dom vitest jsdom
```

- [ ] **Step 4: Configure environment variables**

Create `.env.local`:
```
NEXT_PUBLIC_API_URL=http://localhost:8080
```

- [ ] **Step 5: Update tsconfig.json for path aliases**

Add to `compilerOptions`:
```json
"paths": {
  "@/*": ["./src/*"]
}
```

- [ ] **Step 6: Commit**

```bash
git init
git add .
git commit -m "chore: initialize Next.js 14 project with shadcn/ui"
```

---

### Task 2: TypeScript Types

**Files:**
- Create: `src/types/api.ts`

- [ ] **Step 1: Define all API types**

```typescript
export interface User {
  id: number
  username: string
  name: string
  role: string
  created_at: string
}

export interface Product {
  id: number
  name: string
  price: number
  stock: number
  category_id: number
  category_name?: string
}

export interface Category {
  id: number
  name: string
  description: string
}

export interface CheckoutItem {
  product_id: number
  quantity: number
}

export interface TransactionDetail {
  id: number
  transaction_id: number
  product_id: number
  product_name: string
  quantity: number
  subtotal: number
}

export interface Transaction {
  id: number
  total_amount: number
  created_at: string
  details: TransactionDetail[]
}

export interface SalesSummary {
  total_revenue: number
  total_transactions: number
  top_product?: {
    name: string
    qty_sold: number
  }
}

export interface LoginInput {
  username: string
  password: string
}

export interface LoginResponse {
  message: string
  token: string
  user: User
}
```

- [ ] **Step 2: Commit**

```bash
git add src/types/api.ts
git commit -m "feat: add TypeScript types for API models"
```

---

### Task 3: API Client Setup

**Files:**
- Create: `src/lib/api-client.ts`
- Create: `src/lib/utils.ts` (if not exists from shadcn)

- [ ] **Step 1: Create API client with auth header**

```typescript
const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

function getToken(): string | null {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('token')
  }
  return null
}

async function fetchWithAuth(url: string, options: RequestInit = {}) {
  const token = getToken()
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options.headers,
  }
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  
  const response = await fetch(`${API_URL}${url}`, {
    ...options,
    headers,
  })
  
  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || `HTTP ${response.status}`)
  }
  
  if (response.status === 204) {
    return null
  }
  
  return response.json()
}

export const apiClient = {
  get: (url: string) => fetchWithAuth(url, { method: 'GET' }),
  post: (url: string, data: unknown) => fetchWithAuth(url, { method: 'POST', body: JSON.stringify(data) }),
  put: (url: string, data: unknown) => fetchWithAuth(url, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (url: string) => fetchWithAuth(url, { method: 'DELETE' }),
}
```

- [ ] **Step 2: Commit**

```bash
git add src/lib/api-client.ts
git commit -m "feat: add API client with JWT auth header"
```

---

### Task 4: TanStack Query Setup

**Files:**
- Create: `src/lib/query-client.ts`
- Create: `src/components/providers/QueryProvider.tsx`

- [ ] **Step 1: Create QueryClient config**

```typescript
import { QueryClient } from '@tanstack/react-query'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      refetchOnWindowFocus: true,
      retry: 1,
    },
  },
})
```

- [ ] **Step 2: Create QueryProvider component**

```typescript
'use client'

import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { queryClient } from '@/lib/query-client'

export function QueryProvider({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}
```

- [ ] **Step 3: Wrap root layout**

Modify `src/app/layout.tsx`:
```typescript
import { QueryProvider } from '@/components/providers/QueryProvider'

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <QueryProvider>{children}</QueryProvider>
      </body>
    </html>
  )
}
```

- [ ] **Step 4: Commit**

```bash
git add src/lib/query-client.ts src/components/providers/QueryProvider.tsx src/app/layout.tsx
git commit -m "feat: setup TanStack Query with QueryProvider"
```

---

### Task 5: Zustand Auth Store

**Files:**
- Create: `src/stores/auth.store.ts`
- Create: `src/hooks/useAuth.ts`

- [ ] **Step 1: Create auth store**

```typescript
import { create } from 'zustand'
import { User } from '@/types/api'

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  setAuth: (token: string, user: User) => void
  logout: () => void
  initAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isAuthenticated: false,
  
  setAuth: (token, user) => {
    localStorage.setItem('token', token)
    set({ token, user, isAuthenticated: true })
  },
  
  logout: () => {
    localStorage.removeItem('token')
    set({ token: null, user: null, isAuthenticated: false })
  },
  
  initAuth: () => {
    const token = localStorage.getItem('token')
    if (token) {
      set({ token, isAuthenticated: true })
    }
  },
}))
```

- [ ] **Step 2: Create useAuth hook**

```typescript
import { useAuthStore } from '@/stores/auth.store'
import { apiClient } from '@/lib/api-client'
import { LoginInput, LoginResponse } from '@/types/api'
import { useMutation, useQuery } from '@tanstack/react-query'

export function useAuth() {
  const { user, isAuthenticated, setAuth, logout } = useAuthStore()
  
  const loginMutation = useMutation({
    mutationFn: async (data: LoginInput) => {
      const response: LoginResponse = await apiClient.post('/api/auth/login', data)
      return response
    },
    onSuccess: (data) => {
      setAuth(data.token, data.user)
    },
  })
  
  const { data: meData } = useQuery({
    queryKey: ['me'],
    queryFn: () => apiClient.get('/api/auth/me'),
    enabled: isAuthenticated && !user,
  })
  
  return {
    user,
    isAuthenticated,
    login: loginMutation.mutate,
    isLoggingIn: loginMutation.isPending,
    loginError: loginMutation.error,
    logout,
  }
}
```

- [ ] **Step 3: Commit**

```bash
git add src/stores/auth.store.ts src/hooks/useAuth.ts
git commit -m "feat: add Zustand auth store and useAuth hook"
```

---

### Task 6: Login Page

**Files:**
- Create: `src/app/auth/login/page.tsx`
- Create: `src/validations/schemas.ts`

- [ ] **Step 1: Add shadcn form components**

```bash
npx shadcn-ui@latest add form input button
```

- [ ] **Step 2: Create validation schema**

```typescript
import { z } from 'zod'

export const loginSchema = z.object({
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
})

export type LoginFormData = z.infer<typeof loginSchema>
```

- [ ] **Step 3: Create login page**

```typescript
'use client'

import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import { loginSchema, LoginFormData } from '@/validations/schemas'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

export default function LoginPage() {
  const router = useRouter()
  const { login, isLoggingIn, loginError } = useAuth()
  
  const form = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: '', password: '' },
  })
  
  const onSubmit = (data: LoginFormData) => {
    login(data, {
      onSuccess: () => router.push('/dashboard'),
    })
  }
  
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="w-full max-w-md space-y-6 p-6">
        <h1 className="text-2xl font-bold text-center">Kasir Login</h1>
        
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="username"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Username</FormLabel>
                  <FormControl>
                    <Input placeholder="admin" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Password</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="••••••" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            
            {loginError && (
              <p className="text-sm text-red-500">{loginError.message}</p>
            )}
            
            <Button type="submit" className="w-full" disabled={isLoggingIn}>
              {isLoggingIn ? 'Logging in...' : 'Login'}
            </Button>
          </form>
        </Form>
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Commit**

```bash
git add src/app/auth/login/page.tsx src/validations/schemas.ts
git commit -m "feat: add login page with form validation"
```

---

### Task 7: Dashboard Layout + Sidebar

**Files:**
- Create: `src/app/dashboard/layout.tsx`
- Create: `src/components/layout/Sidebar.tsx`
- Create: `src/components/layout/Header.tsx`

- [ ] **Step 1: Add shadcn components**

```bash
npx shadcn-ui@latest add sheet avatar dropdown-menu
```

- [ ] **Step 2: Create Sidebar component**

```typescript
'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { LayoutDashboard, ShoppingCart, Package, Tags, BarChart3, Receipt, Settings } from 'lucide-react'
import { cn } from '@/lib/utils'

const navItems = [
  { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/pos', label: 'POS', icon: ShoppingCart },
  { href: '/dashboard/products', label: 'Products', icon: Package },
  { href: '/dashboard/categories', label: 'Categories', icon: Tags },
  { href: '/dashboard/reports', label: 'Reports', icon: BarChart3 },
  { href: '/dashboard/transactions', label: 'Transactions', icon: Receipt },
  { href: '/dashboard/settings', label: 'Settings', icon: Settings },
]

export function Sidebar() {
  const pathname = usePathname()
  
  return (
    <aside className="w-64 h-screen border-r bg-background">
      <div className="p-6">
        <h1 className="text-xl font-bold">Kasir App</h1>
      </div>
      
      <nav className="px-4 space-y-1">
        {navItems.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={cn(
              'flex items-center gap-3 px-4 py-2 rounded-lg text-sm font-medium transition-colors',
              pathname === item.href
                ? 'bg-primary text-primary-foreground'
                : 'hover:bg-muted'
            )}
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </Link>
        ))}
      </nav>
    </aside>
  )
}
```

- [ ] **Step 3: Create Header component**

```typescript
'use client'

import { useAuth } from '@/hooks/useAuth'
import { Button } from '@/components/ui/button'
import { LogOut } from 'lucide-react'

export function Header() {
  const { user, logout } = useAuth()
  
  return (
    <header className="h-16 border-b bg-background flex items-center justify-between px-6">
      <div />
      
      <div className="flex items-center gap-4">
        <span className="text-sm">{user?.name}</span>
        <Button variant="ghost" size="sm" onClick={logout}>
          <LogOut className="h-4 w-4 mr-2" />
          Logout
        </Button>
      </div>
    </header>
  )
}
```

- [ ] **Step 4: Create dashboard layout**

```typescript
import { Sidebar } from '@/components/layout/Sidebar'
import { Header } from '@/components/layout/Header'

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen">
      <Sidebar />
      
      <div className="flex-1 flex flex-col overflow-hidden">
        <Header />
        
        <main className="flex-1 overflow-auto p-6">
          {children}
        </main>
      </div>
    </div>
  )
}
```

- [ ] **Step 5: Commit**

```bash
git add src/components/layout/Sidebar.tsx src/components/layout/Header.tsx src/app/dashboard/layout.tsx
git commit -m "feat: add dashboard layout with sidebar and header"
```

---

### Task 8: Dashboard Home Page

**Files:**
- Create: `src/hooks/api/useDashboard.ts`
- Create: `src/app/dashboard/page.tsx`

- [ ] **Step 1: Create dashboard API hook**

```typescript
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'
import { SalesSummary } from '@/types/api'

export function useTodayReport() {
  return useQuery<SalesSummary>({
    queryKey: ['report', 'today'],
    queryFn: () => apiClient.get('/api/report/hari-ini'),
    staleTime: 60 * 1000, // 1 minute
  })
}
```

- [ ] **Step 2: Create dashboard page**

```typescript
'use client'

import { useTodayReport } from '@/hooks/api/useDashboard'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DollarSign, ShoppingCart, TrendingUp } from 'lucide-react'

export default function DashboardPage() {
  const { data: report, isLoading } = useTodayReport()
  
  if (isLoading) {
    return <div>Loading...</div>
  }
  
  return (
    <div className="space-y-6">
      <h2 className="text-3xl font-bold">Dashboard</h2>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Today's Revenue</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              Rp {report?.total_revenue.toLocaleString()}
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Transactions</CardTitle>
            <ShoppingCart className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{report?.total_transactions}</div>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Top Product</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{report?.top_product?.name || '-'}</div>
            <p className="text-xs text-muted-foreground">
              {report?.top_product?.qty_sold} sold
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
```

- [ ] **Step 3: Commit**

```bash
git add src/hooks/api/useDashboard.ts src/app/dashboard/page.tsx
git commit -m "feat: add dashboard home with today's report"
```

---

### Task 9: POS (Cashier) Page

**Files:**
- Create: `src/stores/cart.store.ts`
- Create: `src/hooks/api/useProducts.ts`
- Create: `src/components/pos/ProductGrid.tsx`
- Create: `src/components/pos/CartSidebar.tsx`
- Create: `src/app/pos/page.tsx`

- [ ] **Step 1: Create cart store**

```typescript
import { create } from 'zustand'
import { Product } from '@/types/api'

interface CartItem {
  product: Product
  quantity: number
}

interface CartState {
  items: CartItem[]
  addItem: (product: Product) => void
  removeItem: (productId: number) => void
  updateQuantity: (productId: number, quantity: number) => void
  clear: () => void
  total: number
  itemCount: number
}

export const useCartStore = create<CartState>((set, get) => ({
  items: [],
  
  addItem: (product) => {
    const items = get().items
    const existing = items.find((item) => item.product.id === product.id)
    
    if (existing) {
      set({
        items: items.map((item) =>
          item.product.id === product.id
            ? { ...item, quantity: item.quantity + 1 }
            : item
        ),
      })
    } else {
      set({ items: [...items, { product, quantity: 1 }] })
    }
  },
  
  removeItem: (productId) => {
    set({ items: get().items.filter((item) => item.product.id !== productId) })
  },
  
  updateQuantity: (productId, quantity) => {
    if (quantity <= 0) {
      get().removeItem(productId)
      return
    }
    
    set({
      items: get().items.map((item) =>
        item.product.id === productId ? { ...item, quantity } : item
      ),
    })
  },
  
  clear: () => set({ items: [] }),
  
  get total() {
    return get().items.reduce((sum, item) => sum + item.product.price * item.quantity, 0)
  },
  
  get itemCount() {
    return get().items.reduce((count, item) => count + item.quantity, 0)
  },
}))
```

- [ ] **Step 2: Create products API hook**

```typescript
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'
import { Product } from '@/types/api'

export function useProducts() {
  return useQuery<Product[]>({
    queryKey: ['products'],
    queryFn: () => apiClient.get('/api/products'),
  })
}
```

- [ ] **Step 3: Create ProductGrid component**

```typescript
'use client'

import { useProducts } from '@/hooks/api/useProducts'
import { useCartStore } from '@/stores/cart.store'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useState } from 'react'

export function ProductGrid() {
  const [search, setSearch] = useState('')
  const { data: products, isLoading } = useProducts()
  const addItem = useCartStore((state) => state.addItem)
  
  const filtered = products?.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase())
  )
  
  if (isLoading) return <div>Loading products...</div>
  
  return (
    <div className="space-y-4">
      <Input
        placeholder="Search products..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />
      
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {filtered?.map((product) => (
          <Card
            key={product.id}
            className="cursor-pointer hover:shadow-md transition-shadow"
            onClick={() => addItem(product)}
          >
            <CardContent className="p-4">
              <div className="aspect-square bg-muted rounded-md mb-3" />
              <h3 className="font-medium truncate">{product.name}</h3>
              <p className="text-sm text-muted-foreground">
                Rp {product.price.toLocaleString()}
              </p>
              <p className="text-xs text-muted-foreground">Stock: {product.stock}</p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Create CartSidebar component**

```typescript
'use client'

import { useCartStore } from '@/stores/cart.store'
import { Button } from '@/components/ui/button'
import { Minus, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { apiClient } from '@/lib/api-client'
import { useMutation } from '@tanstack/react-query'

export function CartSidebar() {
  const { items, updateQuantity, removeItem, clear, total } = useCartStore()
  const [isCheckingOut, setIsCheckingOut] = useState(false)
  
  const checkout = useMutation({
    mutationFn: () =>
      apiClient.post('/api/checkout', {
        items: items.map((item) => ({
          product_id: item.product.id,
          quantity: item.quantity,
        })),
      }),
    onSuccess: () => {
      clear()
      alert('Checkout successful!')
    },
  })
  
  return (
    <div className="w-96 h-full border-l bg-background flex flex-col">
      <div className="p-4 border-b">
        <h2 className="font-bold text-lg">Cart</h2>
      </div>
      
      <div className="flex-1 overflow-auto p-4 space-y-4">
        {items.length === 0 ? (
          <p className="text-muted-foreground text-center">Cart is empty</p>
        ) : (
          items.map((item) => (
            <div key={item.product.id} className="flex items-center gap-3">
              <div className="flex-1">
                <p className="font-medium">{item.product.name}</p>
                <p className="text-sm text-muted-foreground">
                  Rp {item.product.price.toLocaleString()}
                </p>
              </div>
              
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => updateQuantity(item.product.id, item.quantity - 1)}
                >
                  <Minus className="h-4 w-4" />
                </Button>
                
                <span className="w-8 text-center">{item.quantity}</span>
                
                <Button
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => updateQuantity(item.product.id, item.quantity + 1)}
                >
                  <Plus className="h-4 w-4" />
                </Button>
                
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-red-500"
                  onClick={() => removeItem(item.product.id)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))
        )}
      </div>
      
      <div className="p-4 border-t space-y-4">
        <div className="flex justify-between text-lg font-bold">
          <span>Total</span>
          <span>Rp {total.toLocaleString()}</span>
        </div>
        
        <Button
          className="w-full"
          disabled={items.length === 0 || checkout.isPending}
          onClick={() => checkout.mutate()}
        >
          {checkout.isPending ? 'Processing...' : 'Checkout'}
        </Button>
      </div>
    </div>
  )
}
```

- [ ] **Step 5: Create POS page**

```typescript
'use client'

import { ProductGrid } from '@/components/pos/ProductGrid'
import { CartSidebar } from '@/components/pos/CartSidebar'

export default function POSPage() {
  return (
    <div className="flex h-screen">
      <div className="flex-1 overflow-auto p-6">
        <h1 className="text-2xl font-bold mb-6">Point of Sale</h1>
        <ProductGrid />
      </div>
      
      <CartSidebar />
    </div>
  )
}
```

- [ ] **Step 6: Commit**

```bash
git add src/stores/cart.store.ts src/hooks/api/useProducts.ts src/components/pos/ProductGrid.tsx src/components/pos/CartSidebar.tsx src/app/pos/page.tsx
git commit -m "feat: add POS page with product grid and cart"
```

---

### Task 10: Products CRUD

**Files:**
- Create: `src/components/common/DataTable.tsx`
- Create: `src/app/dashboard/products/page.tsx`
- Create: `src/components/products/ProductForm.tsx`
- Create: `src/app/dashboard/products/new/page.tsx`

- [ ] **Step 1: Add shadcn components**

```bash
npx shadcn-ui@latest add table dialog alert-dialog
```

- [ ] **Step 2: Create DataTable component**

```typescript
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

interface Column<T> {
  key: string
  header: string
  cell: (row: T) => React.ReactNode
}

interface DataTableProps<T> {
  columns: Column<T>[]
  data: T[]
  isLoading?: boolean
}

export function DataTable<T>({ columns, data, isLoading }: DataTableProps<T>) {
  if (isLoading) return <div>Loading...</div>
  
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((col) => (
              <TableHead key={col.key}>{col.header}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row, i) => (
            <TableRow key={i}>
              {columns.map((col) => (
                <TableCell key={col.key}>{col.cell(row)}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
```

- [ ] **Step 3: Create products page**

```typescript
'use client'

import Link from 'next/link'
import { useProducts } from '@/hooks/api/useProducts'
import { DataTable } from '@/components/common/DataTable'
import { Button } from '@/components/ui/button'
import { Product } from '@/types/api'

export default function ProductsPage() {
  const { data: products, isLoading } = useProducts()
  
  const columns = [
    { key: 'name', header: 'Name', cell: (p: Product) => p.name },
    { key: 'price', header: 'Price', cell: (p: Product) => `Rp ${p.price.toLocaleString()}` },
    { key: 'stock', header: 'Stock', cell: (p: Product) => p.stock },
    { key: 'category', header: 'Category', cell: (p: Product) => p.category_name || '-' },
  ]
  
  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">Products</h2>
        <Link href="/dashboard/products/new">
          <Button>Add Product</Button>
        </Link>
      </div>
      
      <DataTable columns={columns} data={products || []} isLoading={isLoading} />
    </div>
  )
}
```

- [ ] **Step 4: Create product form (simplified)**

```typescript
'use client'

import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useRouter } from 'next/navigation'

const productSchema = z.object({
  name: z.string().min(1),
  price: z.coerce.number().min(0),
  stock: z.coerce.number().min(0),
  category_id: z.coerce.number().min(1),
})

type ProductFormData = z.infer<typeof productSchema>

export function ProductForm() {
  const router = useRouter()
  const form = useForm<ProductFormData>({
    resolver: zodResolver(productSchema),
    defaultValues: { name: '', price: 0, stock: 0, category_id: 1 },
  })
  
  const createProduct = useMutation({
    mutationFn: (data: ProductFormData) => apiClient.post('/api/products', data),
    onSuccess: () => {
      router.push('/dashboard/products')
    },
  })
  
  return (
    <form onSubmit={form.handleSubmit((data) => createProduct.mutate(data))} className="space-y-4 max-w-md">
      <div>
        <label className="text-sm font-medium">Name</label>
        <Input {...form.register('name')} />
      </div>
      
      <div>
        <label className="text-sm font-medium">Price</label>
        <Input type="number" {...form.register('price')} />
      </div>
      
      <div>
        <label className="text-sm font-medium">Stock</label>
        <Input type="number" {...form.register('stock')} />
      </div>
      
      <div>
        <label className="text-sm font-medium">Category ID</label>
        <Input type="number" {...form.register('category_id')} />
      </div>
      
      <Button type="submit" disabled={createProduct.isPending}>
        {createProduct.isPending ? 'Creating...' : 'Create Product'}
      </Button>
    </form>
  )
}
```

- [ ] **Step 5: Create new product page**

```typescript
import { ProductForm } from '@/components/products/ProductForm'

export default function NewProductPage() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">New Product</h2>
      <ProductForm />
    </div>
  )
}
```

- [ ] **Step 6: Commit**

```bash
git add src/components/common/DataTable.tsx src/app/dashboard/products/page.tsx src/components/products/ProductForm.tsx src/app/dashboard/products/new/page.tsx
git commit -m "feat: add products CRUD pages"
```

---

### Task 11: Categories, Reports, Transactions, Settings

**Files:**
- Create: `src/app/dashboard/categories/page.tsx`
- Create: `src/app/dashboard/reports/page.tsx`
- Create: `src/app/dashboard/transactions/page.tsx`
- Create: `src/app/dashboard/settings/page.tsx`

- [ ] **Step 1: Create categories page**

```typescript
'use client'

import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'
import { DataTable } from '@/components/common/DataTable'
import { Category } from '@/types/api'

export default function CategoriesPage() {
  const { data: categories, isLoading } = useQuery<Category[]>({
    queryKey: ['categories'],
    queryFn: () => apiClient.get('/api/categories'),
  })
  
  const columns = [
    { key: 'name', header: 'Name', cell: (c: Category) => c.name },
    { key: 'description', header: 'Description', cell: (c: Category) => c.description },
  ]
  
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Categories</h2>
      <DataTable columns={columns} data={categories || []} isLoading={isLoading} />
    </div>
  )
}
```

- [ ] **Step 2: Create reports page**

```typescript
'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { SalesSummary } from '@/types/api'

export default function ReportsPage() {
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  
  const { data: todayReport } = useQuery<SalesSummary>({
    queryKey: ['report', 'today'],
    queryFn: () => apiClient.get('/api/report/hari-ini'),
  })
  
  const { data: rangeReport, refetch } = useQuery<SalesSummary>({
    queryKey: ['report', startDate, endDate],
    queryFn: () => apiClient.get(`/api/report?start_date=${startDate}&end_date=${endDate}`),
    enabled: false,
  })
  
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Reports</h2>
      
      <Card>
        <CardHeader>
          <CardTitle>Today</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-3 gap-4">
          <div>
            <p className="text-sm text-muted-foreground">Revenue</p>
            <p className="text-2xl font-bold">Rp {todayReport?.total_revenue.toLocaleString()}</p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Transactions</p>
            <p className="text-2xl font-bold">{todayReport?.total_transactions}</p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Top Product</p>
            <p className="text-2xl font-bold">{todayReport?.top_product?.name || '-'}</p>
          </div>
        </CardContent>
      </Card>
      
      <div className="flex gap-4 items-end">
        <div>
          <label className="text-sm font-medium">Start Date</label>
          <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
        </div>
        <div>
          <label className="text-sm font-medium">End Date</label>
          <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
        </div>
        <Button onClick={() => refetch()}>Generate Report</Button>
      </div>
      
      {rangeReport && (
        <Card>
          <CardContent className="pt-6 grid grid-cols-3 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Revenue</p>
              <p className="text-2xl font-bold">Rp {rangeReport.total_revenue.toLocaleString()}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Transactions</p>
              <p className="text-2xl font-bold">{rangeReport.total_transactions}</p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
```

- [ ] **Step 3: Create transactions page**

```typescript
'use client'

import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'
import { DataTable } from '@/components/common/DataTable'
import { Transaction } from '@/types/api'

export default function TransactionsPage() {
  // Note: Backend doesn't have a list transactions endpoint yet
  // This is a placeholder structure
  const { data: transactions, isLoading } = useQuery<Transaction[]>({
    queryKey: ['transactions'],
    queryFn: () => apiClient.get('/api/transactions'),
    // This will fail until backend adds the endpoint
  })
  
  const columns = [
    { key: 'id', header: 'ID', cell: (t: Transaction) => t.id },
    { key: 'total', header: 'Total', cell: (t: Transaction) => `Rp ${t.total_amount.toLocaleString()}` },
    { key: 'date', header: 'Date', cell: (t: Transaction) => new Date(t.created_at).toLocaleDateString() },
    { key: 'items', header: 'Items', cell: (t: Transaction) => t.details.length },
  ]
  
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Transactions</h2>
      <p className="text-muted-foreground">Transaction history (requires backend endpoint)</p>
      <DataTable columns={columns} data={transactions || []} isLoading={isLoading} />
    </div>
  )
}
```

- [ ] **Step 4: Create settings page**

```typescript
'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Settings</h2>
      
      <Card>
        <CardHeader>
          <CardTitle>Application Info</CardTitle>
        </CardHeader>
        <CardContent>
          <p>Kasir POS System v1.0</p>
          <p className="text-sm text-muted-foreground">Connected to go-kasir-api backend</p>
        </CardContent>
      </Card>
    </div>
  )
}
```

- [ ] **Step 5: Commit**

```bash
git add src/app/dashboard/categories/page.tsx src/app/dashboard/reports/page.tsx src/app/dashboard/transactions/page.tsx src/app/dashboard/settings/page.tsx
git commit -m "feat: add categories, reports, transactions, settings pages"
```

---

### Task 12: Auth Protection + Final Polish

**Files:**
- Create: `src/components/providers/AuthProvider.tsx`
- Modify: `src/app/layout.tsx`

- [ ] **Step 1: Create AuthProvider**

```typescript
'use client'

import { useEffect } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { useAuthStore } from '@/stores/auth.store'

const publicPaths = ['/auth/login']

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { isAuthenticated, initAuth } = useAuthStore()
  
  useEffect(() => {
    initAuth()
  }, [initAuth])
  
  useEffect(() => {
    if (!isAuthenticated && !publicPaths.includes(pathname)) {
      router.push('/auth/login')
    }
    if (isAuthenticated && pathname === '/auth/login') {
      router.push('/dashboard')
    }
  }, [isAuthenticated, pathname, router])
  
  return <>{children}</>
}
```

- [ ] **Step 2: Update root layout**

```typescript
import { QueryProvider } from '@/components/providers/QueryProvider'
import { AuthProvider } from '@/components/providers/AuthProvider'
import './globals.css'

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <QueryProvider>
          <AuthProvider>{children}</AuthProvider>
        </QueryProvider>
      </body>
    </html>
  )
}
```

- [ ] **Step 3: Add redirect from root**

Modify `src/app/page.tsx`:
```typescript
import { redirect } from 'next/navigation'

export default function HomePage() {
  redirect('/dashboard')
}
```

- [ ] **Step 4: Commit**

```bash
git add src/components/providers/AuthProvider.tsx src/app/layout.tsx src/app/page.tsx
git commit -m "feat: add auth protection and redirects"
```

---

## Execution Handoff

**Plan complete.** Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
