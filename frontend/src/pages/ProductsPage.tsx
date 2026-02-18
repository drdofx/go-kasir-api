import { useState, useEffect, useCallback } from 'react'
import { Plus, Pencil, Trash2, X } from 'lucide-react'
import { productApi, categoryApi } from '../api'
import { useToast } from '../toast-context'
import { money } from '../utils'
import type { Product, ProductInput, Category } from '../types'

function emptyInput(): ProductInput {
    return { name: '', price: 0, stock: 0, category_id: 0 }
}

export default function ProductsPage() {
    const [products, setProducts] = useState<Product[]>([])
    const [categories, setCategories] = useState<Category[]>([])
    const [modal, setModal] = useState<'create' | 'edit' | null>(null)
    const [editing, setEditing] = useState<Product | null>(null)
    const [form, setForm] = useState<ProductInput>(emptyInput())
    const [saving, setSaving] = useState(false)
    const toast = useToast()

    const load = useCallback(async () => {
        try {
            const [p, c] = await Promise.all([productApi.list(), categoryApi.list()])
            setProducts(Array.isArray(p) ? p : [])
            setCategories(Array.isArray(c) ? c : [])
        } catch { toast.error('Failed to load products') }
    }, [toast])

    useEffect(() => { load() }, [load])

    const openCreate = () => { setForm(emptyInput()); setEditing(null); setModal('create') }
    const openEdit = (p: Product) => {
        setEditing(p)
        setForm({ name: p.name, price: p.price, stock: p.stock, category_id: p.category_id })
        setModal('edit')
    }

    const save = async () => {
        if (!form.name.trim()) { toast.warning('Name is required'); return }
        setSaving(true)
        try {
            if (modal === 'create') {
                await productApi.create(form)
                toast.success('Product created')
            } else if (editing) {
                await productApi.update(editing.id, form)
                toast.success('Product updated')
            }
            setModal(null)
            load()
        } catch { toast.error('Failed to save product') }
        setSaving(false)
    }

    const remove = async (p: Product) => {
        if (!confirm(`Delete "${p.name}"?`)) return
        try {
            await productApi.delete(p.id)
            toast.success('Product deleted')
            load()
        } catch { toast.error('Failed to delete product') }
    }

    return (
        <>
            <div className="page-header">
                <h2>Products</h2>
                <p>Manage your product inventory</p>
            </div>

            <div className="card">
                <div className="card-header">
                    <h3>All Products ({products.length})</h3>
                    <button className="btn-primary btn-sm" onClick={openCreate}><Plus size={14} /> Add Product</button>
                </div>

                <div style={{ overflowX: 'auto' }}>
                    <table className="data-table">
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Name</th>
                                <th>Category</th>
                                <th>Price</th>
                                <th>Stock</th>
                                <th style={{ width: 100 }}>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {products.length === 0 ? (
                                <tr><td colSpan={6} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: 40 }}>No products</td></tr>
                            ) : products.map(p => (
                                <tr key={p.id}>
                                    <td style={{ color: 'var(--text-muted)' }}>#{p.id}</td>
                                    <td style={{ fontWeight: 600 }}>{p.name}</td>
                                    <td>{p.category_name || '-'}</td>
                                    <td style={{ color: 'var(--accent)', fontWeight: 600 }}>{money(p.price)}</td>
                                    <td>
                                        <span className={`badge ${p.stock > 0 ? 'badge-stock' : 'badge-out'}`}>
                                            {p.stock}
                                        </span>
                                    </td>
                                    <td>
                                        <div style={{ display: 'flex', gap: 6 }}>
                                            <button className="btn-icon btn" onClick={() => openEdit(p)}><Pencil size={14} /></button>
                                            <button className="btn-icon btn-danger" onClick={() => remove(p)}><Trash2 size={14} /></button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>

            {modal && (
                <div className="modal-overlay" onClick={() => setModal(null)}>
                    <div className="modal" onClick={e => e.stopPropagation()}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <h3>{modal === 'create' ? 'New Product' : 'Edit Product'}</h3>
                            <button className="btn-icon btn" onClick={() => setModal(null)}><X size={16} /></button>
                        </div>
                        <div className="form-group">
                            <label>Name</label>
                            <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} placeholder="Product name" />
                        </div>
                        <div className="form-row">
                            <div className="form-group">
                                <label>Price</label>
                                <input type="number" value={form.price || ''} onChange={e => setForm({ ...form, price: Number(e.target.value) })} placeholder="0" />
                            </div>
                            <div className="form-group">
                                <label>Stock</label>
                                <input type="number" value={form.stock || ''} onChange={e => setForm({ ...form, stock: Number(e.target.value) })} placeholder="0" />
                            </div>
                        </div>
                        <div className="form-group">
                            <label>Category</label>
                            <select value={form.category_id || ''} onChange={e => setForm({ ...form, category_id: Number(e.target.value) })}>
                                <option value="">Select category</option>
                                {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                            </select>
                        </div>
                        <div className="modal-actions">
                            <button className="btn" onClick={() => setModal(null)}>Cancel</button>
                            <button className="btn-primary" onClick={save} disabled={saving}>
                                {saving ? 'Saving...' : modal === 'create' ? 'Create' : 'Update'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </>
    )
}
