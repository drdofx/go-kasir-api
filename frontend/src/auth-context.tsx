import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import { authApi } from './api'

interface User {
    id: number
    username: string
    name: string
    role: string
}

interface AuthCtx {
    user: User | null
    loading: boolean
    login: (username: string, password: string) => Promise<void>
    logout: () => Promise<void>
    isAuthenticated: boolean
}

const Ctx = createContext<AuthCtx | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null)
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        authApi.me()
            .then(setUser)
            .catch(() => setUser(null))
            .finally(() => setLoading(false))
    }, [])

    const login = useCallback(async (username: string, password: string) => {
        const res = await authApi.login(username, password)
        setUser(res.user)
    }, [])

    const logout = useCallback(async () => {
        await authApi.logout()
        setUser(null)
    }, [])

    return (
        <Ctx.Provider value={{ user, loading, login, logout, isAuthenticated: !!user }}>
            {children}
        </Ctx.Provider>
    )
}

export function useAuth() {
    const ctx = useContext(Ctx)
    if (!ctx) throw new Error('useAuth must be inside AuthProvider')
    return ctx
}
