import { useState, useEffect, useCallback } from 'react'
import { Search, ShoppingBag, Trash2, Minus, Plus, CreditCard, Loader } from 'lucide-react'
import { productApi, categoryApi, transactionApi } from '../api'
import { useCart } from '../cart-context'
import { useToast } from '../toast-context'
import { money } from '../utils'
import type { Product, Category } from '../types'

export default function PosPage() {
    const [products, setProducts] = useState<Product[]>([])
    const [categories, setCategories] = useState<Category[]>([])
    const [search, setSearch] = useState('')
    const [catFilter, setCatFilter] = useState<number | ''>('')
    const [loading, setLoading] = useState(false)
    const [checkingOut, setCheckingOut] = useState(false)
    const cart = useCart()
    const toast = useToast()

    const load = useCallback(async () => {
        setLoading(true)
        try {
            const [prods, cats] = await Promise.all([
                productApi.list(search || undefined),
                categoryApi.list(),
            ])
            setProducts(Array.isArray(prods) ? prods : [])
            setCategories(Array.isArray(cats) ? cats : [])
        } catch { toast.error('Failed to load products') }
        setLoading(false)
    }, [search, toast])

    useEffect(() => { load() }, [load])

    const filtered = catFilter
        ? products.filter(p => p.category_id === catFilter)
        : products

    const handleCheckout = async () => {
        if (!cart.items.length) { toast.warning('Cart is empty'); return }
        setCheckingOut(true)
        try {
            const trx = await transactionApi.checkout({
                items: cart.items.map(i => ({ product_id: i.product.id, quantity: i.qty })),
            })
            toast.success(`Checkout success! Transaction #${trx.id} — ${money(trx.total_amount)}`)
            cart.clear()
            load()
        } catch { toast.error('Checkout failed') }
        setCheckingOut(false)
    }

    return (
        <>
            {/* Checkout Loading Overlay */}
            {checkingOut && (
                <div className="checkout-overlay">
                    <div className="checkout-overlay-content">
                        <Loader size={40} className="spin" />
                        <p>Processing transaction...</p>
                    </div>
                </div>
            )}

            <div className="page-header">
                <h2>Point of Sale</h2>
                <p>Browse products and process transactions</p>
            </div>

            <div className="pos-layout">
                {/* Product Grid */}
                <div>
                    <div className="pos-filters">
                        <div className="search-input" style={{ position: 'relative' }}>
                            <Search style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', width: 16, height: 16, color: 'var(--text-muted)' }} />
                            <input
                                placeholder="Search products..."
                                value={search}
                                onChange={e => setSearch(e.target.value)}
                                style={{ paddingLeft: 36 }}
                                disabled={checkingOut}
                            />
                        </div>
                        <select
                            value={catFilter}
                            onChange={e => setCatFilter(e.target.value ? Number(e.target.value) : '')}
                            style={{ width: 160 }}
                            disabled={checkingOut}
                        >
                            <option value="">All Categories</option>
                            {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                        </select>
                    </div>

                    {loading ? (
                        <div className="empty-state"><p>Loading products...</p></div>
                    ) : filtered.length === 0 ? (
                        <div className="empty-state">
                            <ShoppingBag size={48} />
                            <p>No products found</p>
                        </div>
                    ) : (
                        <div className={`product-grid${checkingOut ? ' disabled-grid' : ''}`}>
                            {filtered.map(p => (
                                <div key={p.id} className="product-card" onClick={() => !checkingOut && p.stock > 0 && cart.add(p)}>
                                    <h4>{p.name}</h4>
                                    <div className="product-meta">{p.category_name || 'Uncategorized'}</div>
                                    <div className="product-meta">
                                        <span className={`badge ${p.stock > 0 ? 'badge-stock' : 'badge-out'}`}>
                                            {p.stock > 0 ? `${p.stock} in stock` : 'Out of stock'}
                                        </span>
                                    </div>
                                    <div className="product-price">{money(p.price)}</div>
                                    <div className="product-actions">
                                        <button
                                            className="btn-primary btn-sm"
                                            disabled={p.stock <= 0 || checkingOut}
                                            onClick={e => { e.stopPropagation(); cart.add(p) }}
                                            style={{ width: '100%', justifyContent: 'center' }}
                                        >
                                            <Plus size={14} /> Add to Cart
                                        </button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Cart Panel */}
                <div className="cart-panel">
                    <div className="card">
                        <div className="card-header">
                            <h3><ShoppingBag size={18} style={{ marginRight: 8, verticalAlign: -3 }} />Cart ({cart.count})</h3>
                            {cart.items.length > 0 && (
                                <button className="btn btn-sm" onClick={cart.clear} disabled={checkingOut}>Clear</button>
                            )}
                        </div>

                        {cart.items.length === 0 ? (
                            <div className="empty-state">
                                <ShoppingBag size={36} />
                                <p>Your cart is empty</p>
                            </div>
                        ) : (
                            <>
                                {cart.items.map(item => (
                                    <div key={item.product.id} className="cart-item">
                                        <div className="cart-item-info">
                                            <h4>{item.product.name}</h4>
                                            <span>{money(item.product.price)} × {item.qty} = {money(item.product.price * item.qty)}</span>
                                        </div>
                                        <div className="cart-qty">
                                            <button onClick={() => cart.setQty(item.product.id, item.qty - 1)} disabled={checkingOut}><Minus size={14} /></button>
                                            <span>{item.qty}</span>
                                            <button onClick={() => cart.setQty(item.product.id, item.qty + 1)} disabled={checkingOut}><Plus size={14} /></button>
                                        </div>
                                        <button className="btn-icon btn-danger" onClick={() => cart.remove(item.product.id)} disabled={checkingOut}>
                                            <Trash2 size={14} />
                                        </button>
                                    </div>
                                ))}

                                <div className="cart-total">
                                    <span className="cart-total-label">Total</span>
                                    <span className="cart-total-value">{money(cart.total)}</span>
                                </div>

                                <button
                                    className="btn-primary"
                                    onClick={handleCheckout}
                                    disabled={checkingOut}
                                    style={{ width: '100%', marginTop: 12, justifyContent: 'center', padding: '14px 20px', fontSize: '0.95rem' }}
                                >
                                    {checkingOut
                                        ? <><Loader size={18} className="spin" /> Processing...</>
                                        : <><CreditCard size={18} /> Checkout</>
                                    }
                                </button>
                            </>
                        )}
                    </div>
                </div>
            </div>
        </>
    )
}
