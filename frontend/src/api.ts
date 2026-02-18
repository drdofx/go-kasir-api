declare global {
    interface Window {
        __APP_CONFIG__?: { apiKey: string }
    }
}

const BASE = ''

function headers(): HeadersInit {
    return { 'Content-Type': 'application/json' }
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
    const res = await fetch(BASE + url, {
        credentials: 'same-origin',
        ...init,
    })
    if (!res.ok) {
        const text = await res.text()
        throw new Error(text || `HTTP ${res.status}`)
    }
    return res.json()
}

// ---------- Auth ----------
export const authApi = {
    login: (username: string, password: string) =>
        request<{ message: string; user: any }>('/api/auth/login', {
            method: 'POST', headers: headers(),
            body: JSON.stringify({ username, password }),
        }),
    logout: () =>
        request<{ message: string }>('/api/auth/logout', { method: 'POST', headers: headers() }),
    me: () =>
        request<any>('/api/auth/me'),
}

// ---------- Products ----------
import type { Product, ProductInput } from './types'

export const productApi = {
    list: (name?: string) =>
        request<Product[]>(name ? `/api/products?name=${encodeURIComponent(name)}` : '/api/products'),
    get: (id: number) => request<Product>(`/api/products/${id}`, { headers: headers() }),
    create: (data: ProductInput) =>
        request<Product>('/api/products', { method: 'POST', headers: headers(), body: JSON.stringify(data) }),
    update: (id: number, data: ProductInput) =>
        request<Product>(`/api/products/${id}`, { method: 'PUT', headers: headers(), body: JSON.stringify(data) }),
    delete: (id: number) =>
        request<{ message: string }>(`/api/products/${id}`, { method: 'DELETE', headers: headers() }),
}

// ---------- Categories ----------
import type { Category, CategoryInput } from './types'

export const categoryApi = {
    list: () => request<Category[]>('/categories'),
    get: (id: number) => request<Category>(`/categories/${id}`),
    create: (data: CategoryInput) =>
        request<Category>('/categories', { method: 'POST', headers: headers(), body: JSON.stringify(data) }),
    update: (id: number, data: CategoryInput) =>
        request<Category>(`/categories/${id}`, { method: 'PUT', headers: headers(), body: JSON.stringify(data) }),
    delete: (id: number) =>
        request<{ message: string }>(`/categories/${id}`, { method: 'DELETE', headers: headers() }),
}

// ---------- Transactions ----------
import type { CheckoutRequest, Transaction, SalesSummary } from './types'

export const transactionApi = {
    checkout: (data: CheckoutRequest) =>
        request<Transaction>('/api/checkout', { method: 'POST', headers: headers(), body: JSON.stringify(data) }),
    todayReport: () => request<SalesSummary>('/api/report/hari-ini'),
    report: (start: string, end: string) =>
        request<SalesSummary>(`/api/report?start_date=${start}&end_date=${end}`),
}
