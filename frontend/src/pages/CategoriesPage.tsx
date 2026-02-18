import { useState, useEffect, useCallback } from 'react'
import { Plus, Pencil, Trash2, X } from 'lucide-react'
import { categoryApi } from '../api'
import { useToast } from '../toast-context'
import type { Category, CategoryInput } from '../types'

function emptyInput(): CategoryInput {
    return { name: '', description: '' }
}

export default function CategoriesPage() {
    const [categories, setCategories] = useState<Category[]>([])
    const [modal, setModal] = useState<'create' | 'edit' | null>(null)
    const [editing, setEditing] = useState<Category | null>(null)
    const [form, setForm] = useState<CategoryInput>(emptyInput())
    const [saving, setSaving] = useState(false)
    const toast = useToast()

    const load = useCallback(async () => {
        try {
            const cats = await categoryApi.list()
            setCategories(Array.isArray(cats) ? cats : [])
        } catch { toast.error('Failed to load categories') }
    }, [toast])

    useEffect(() => { load() }, [load])

    const openCreate = () => { setForm(emptyInput()); setEditing(null); setModal('create') }
    const openEdit = (c: Category) => {
        setEditing(c)
        setForm({ name: c.name, description: c.description })
        setModal('edit')
    }

    const save = async () => {
        if (!form.name.trim()) { toast.warning('Name is required'); return }
        setSaving(true)
        try {
            if (modal === 'create') {
                await categoryApi.create(form)
                toast.success('Category created')
            } else if (editing) {
                await categoryApi.update(editing.id, form)
                toast.success('Category updated')
            }
            setModal(null)
            load()
        } catch { toast.error('Failed to save category') }
        setSaving(false)
    }

    const remove = async (c: Category) => {
        if (!confirm(`Delete "${c.name}"?`)) return
        try {
            await categoryApi.delete(c.id)
            toast.success('Category deleted')
            load()
        } catch { toast.error('Failed to delete category') }
    }

    return (
        <>
            <div className="page-header">
                <h2>Categories</h2>
                <p>Organize your products with categories</p>
            </div>

            <div className="card">
                <div className="card-header">
                    <h3>All Categories ({categories.length})</h3>
                    <button className="btn-primary btn-sm" onClick={openCreate}><Plus size={14} /> Add Category</button>
                </div>

                <table className="data-table">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>Name</th>
                            <th>Description</th>
                            <th style={{ width: 100 }}>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {categories.length === 0 ? (
                            <tr><td colSpan={4} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: 40 }}>No categories</td></tr>
                        ) : categories.map(c => (
                            <tr key={c.id}>
                                <td style={{ color: 'var(--text-muted)' }}>#{c.id}</td>
                                <td style={{ fontWeight: 600 }}>{c.name}</td>
                                <td style={{ color: 'var(--text-secondary)' }}>{c.description || '-'}</td>
                                <td>
                                    <div style={{ display: 'flex', gap: 6 }}>
                                        <button className="btn-icon btn" onClick={() => openEdit(c)}><Pencil size={14} /></button>
                                        <button className="btn-icon btn-danger" onClick={() => remove(c)}><Trash2 size={14} /></button>
                                    </div>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            {modal && (
                <div className="modal-overlay" onClick={() => setModal(null)}>
                    <div className="modal" onClick={e => e.stopPropagation()}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <h3>{modal === 'create' ? 'New Category' : 'Edit Category'}</h3>
                            <button className="btn-icon btn" onClick={() => setModal(null)}><X size={16} /></button>
                        </div>
                        <div className="form-group">
                            <label>Name</label>
                            <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} placeholder="Category name" />
                        </div>
                        <div className="form-group">
                            <label>Description</label>
                            <textarea
                                rows={3}
                                value={form.description}
                                onChange={e => setForm({ ...form, description: e.target.value })}
                                placeholder="Category description"
                            />
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
