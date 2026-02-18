import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import type { Product, CartItem } from './types'

interface CartCtx {
    items: CartItem[]
    add: (p: Product) => void
    remove: (productId: number) => void
    setQty: (productId: number, qty: number) => void
    clear: () => void
    total: number
    count: number
}

const Ctx = createContext<CartCtx | null>(null)

export function CartProvider({ children }: { children: ReactNode }) {
    const [items, setItems] = useState<CartItem[]>([])

    const add = useCallback((p: Product) => {
        setItems(prev => {
            const existing = prev.find(i => i.product.id === p.id)
            if (existing) {
                if (existing.qty >= p.stock) return prev
                return prev.map(i => i.product.id === p.id ? { ...i, qty: i.qty + 1 } : i)
            }
            return [...prev, { product: p, qty: 1 }]
        })
    }, [])

    const remove = useCallback((id: number) => {
        setItems(prev => prev.filter(i => i.product.id !== id))
    }, [])

    const setQty = useCallback((id: number, qty: number) => {
        if (qty <= 0) { remove(id); return }
        setItems(prev => prev.map(i =>
            i.product.id === id ? { ...i, qty: Math.min(qty, i.product.stock) } : i
        ))
    }, [remove])

    const clear = useCallback(() => setItems([]), [])

    const total = items.reduce((s, i) => s + i.product.price * i.qty, 0)
    const count = items.reduce((s, i) => s + i.qty, 0)

    return <Ctx.Provider value={{ items, add, remove, setQty, clear, total, count }}>{children}</Ctx.Provider>
}

export function useCart() {
    const ctx = useContext(Ctx)
    if (!ctx) throw new Error('useCart must be inside CartProvider')
    return ctx
}
