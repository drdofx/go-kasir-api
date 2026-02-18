import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'

interface Toast {
    id: number
    message: string
    type: 'success' | 'error' | 'warning'
}

interface ToastCtx {
    toasts: Toast[]
    success: (msg: string) => void
    error: (msg: string) => void
    warning: (msg: string) => void
}

const Ctx = createContext<ToastCtx | null>(null)
let nextId = 0

export function ToastProvider({ children }: { children: ReactNode }) {
    const [toasts, setToasts] = useState<Toast[]>([])

    const add = useCallback((message: string, type: Toast['type']) => {
        const id = ++nextId
        setToasts(prev => [...prev, { id, message, type }])
        setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 3000)
    }, [])

    return (
        <Ctx.Provider value={{
            toasts,
            success: (m) => add(m, 'success'),
            error: (m) => add(m, 'error'),
            warning: (m) => add(m, 'warning'),
        }}>
            {children}
            <div className="toast-container">
                {toasts.map(t => (
                    <div key={t.id} className={`toast ${t.type}`}>{t.message}</div>
                ))}
            </div>
        </Ctx.Provider>
    )
}

export function useToast() {
    const ctx = useContext(Ctx)
    if (!ctx) throw new Error('useToast must be inside ToastProvider')
    return ctx
}
