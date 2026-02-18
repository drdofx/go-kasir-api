import { useState, useEffect, useCallback } from 'react'
import { DollarSign, ShoppingCart, Trophy, Calendar } from 'lucide-react'
import { transactionApi } from '../api'
import { useToast } from '../toast-context'
import { money } from '../utils'
import type { SalesSummary } from '../types'

export default function ReportsPage() {
    const [today, setToday] = useState<SalesSummary | null>(null)
    const [custom, setCustom] = useState<SalesSummary | null>(null)
    const [startDate, setStartDate] = useState('')
    const [endDate, setEndDate] = useState('')
    const [loading, setLoading] = useState(false)
    const toast = useToast()

    const loadToday = useCallback(async () => {
        try {
            const data = await transactionApi.todayReport()
            setToday(data)
        } catch { toast.error('Failed to load today report') }
    }, [toast])

    useEffect(() => { loadToday() }, [loadToday])

    const loadCustom = async () => {
        if (!startDate || !endDate) { toast.warning('Select both dates'); return }
        setLoading(true)
        try {
            const data = await transactionApi.report(startDate, endDate)
            setCustom(data)
        } catch { toast.error('Failed to load report') }
        setLoading(false)
    }

    const renderStats = (data: SalesSummary, label: string) => (
        <div className="stat-grid" style={{ marginBottom: 24 }}>
            <div className="stat-card">
                <div className="stat-icon orange"><DollarSign size={20} /></div>
                <div className="stat-value">{money(data.total_revenue)}</div>
                <div className="stat-label">{label} Revenue</div>
            </div>
            <div className="stat-card">
                <div className="stat-icon blue"><ShoppingCart size={20} /></div>
                <div className="stat-value">{data.total_transaksi}</div>
                <div className="stat-label">Transactions</div>
            </div>
            {data.produk_terlaris && (
                <div className="stat-card">
                    <div className="stat-icon green"><Trophy size={20} /></div>
                    <div className="stat-value">{data.produk_terlaris.nama}</div>
                    <div className="stat-label">Best Seller — {data.produk_terlaris.qty_terjual} sold</div>
                </div>
            )}
        </div>
    )

    return (
        <>
            <div className="page-header">
                <h2>Sales Reports</h2>
                <p>Track your revenue and best-selling products</p>
            </div>

            {/* Today's Summary */}
            <div className="card" style={{ marginBottom: 20 }}>
                <div className="card-header">
                    <h3><Calendar size={18} style={{ marginRight: 8, verticalAlign: -3 }} />Today's Summary</h3>
                    <button className="btn btn-sm" onClick={loadToday}>Refresh</button>
                </div>
                {today ? renderStats(today, "Today's") : (
                    <div className="empty-state"><p>Loading...</p></div>
                )}
            </div>

            {/* Custom Date Range */}
            <div className="card">
                <div className="card-header">
                    <h3>Custom Date Range</h3>
                </div>
                <div style={{ display: 'flex', gap: 12, alignItems: 'flex-end', marginBottom: 20 }}>
                    <div className="form-group" style={{ flex: 1, marginBottom: 0 }}>
                        <label>Start Date</label>
                        <input type="date" value={startDate} onChange={e => setStartDate(e.target.value)} />
                    </div>
                    <div className="form-group" style={{ flex: 1, marginBottom: 0 }}>
                        <label>End Date</label>
                        <input type="date" value={endDate} onChange={e => setEndDate(e.target.value)} />
                    </div>
                    <button className="btn-primary" onClick={loadCustom} disabled={loading} style={{ marginBottom: 0 }}>
                        {loading ? 'Loading...' : 'Generate'}
                    </button>
                </div>
                {custom && renderStats(custom, 'Period')}
            </div>
        </>
    )
}
