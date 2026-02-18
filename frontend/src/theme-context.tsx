import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'

type Theme = 'dark' | 'light'

interface ThemeCtx {
    theme: Theme
    toggle: () => void
    isDark: boolean
}

const Ctx = createContext<ThemeCtx | null>(null)

function getInitial(): Theme {
    const saved = localStorage.getItem('kasir-theme') as Theme | null
    if (saved === 'dark' || saved === 'light') return saved
    return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
    const [theme, setTheme] = useState<Theme>(getInitial)

    useEffect(() => {
        document.documentElement.setAttribute('data-theme', theme)
        localStorage.setItem('kasir-theme', theme)
    }, [theme])

    const toggle = useCallback(() => {
        setTheme(prev => prev === 'dark' ? 'light' : 'dark')
    }, [])

    return (
        <Ctx.Provider value={{ theme, toggle, isDark: theme === 'dark' }}>
            {children}
        </Ctx.Provider>
    )
}

export function useTheme() {
    const ctx = useContext(Ctx)
    if (!ctx) throw new Error('useTheme must be inside ThemeProvider')
    return ctx
}
