import { useState, type FormEvent } from 'react'
import { useAuth } from '../auth-context'
import { useTheme } from '../theme-context'
import { LogIn, Sun, Moon, Eye, EyeOff } from 'lucide-react'

export default function LoginPage() {
    const { login } = useAuth()
    const { isDark, toggle } = useTheme()
    const [username, setUsername] = useState('')
    const [password, setPassword] = useState('')
    const [showPw, setShowPw] = useState(false)
    const [error, setError] = useState('')
    const [loading, setLoading] = useState(false)

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault()
        setError('')
        setLoading(true)
        try {
            await login(username, password)
        } catch {
            setError('Invalid username or password')
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="login-page">
            <div className="login-card card">
                <div className="login-header">
                    <h1 className="login-brand">Kasir</h1>
                    <p>Point of Sale System</p>
                </div>

                <form onSubmit={handleSubmit}>
                    <div className="form-group">
                        <label htmlFor="username">Username</label>
                        <input
                            id="username"
                            type="text"
                            placeholder="Enter your username"
                            value={username}
                            onChange={e => setUsername(e.target.value)}
                            autoFocus
                            autoComplete="username"
                        />
                    </div>
                    <div className="form-group">
                        <label htmlFor="password">Password</label>
                        <div style={{ position: 'relative' }}>
                            <input
                                id="password"
                                type={showPw ? 'text' : 'password'}
                                placeholder="Enter your password"
                                value={password}
                                onChange={e => setPassword(e.target.value)}
                                autoComplete="current-password"
                                style={{ paddingRight: 40 }}
                            />
                            <button
                                type="button"
                                onClick={() => setShowPw(v => !v)}
                                style={{
                                    position: 'absolute', right: 8, top: '50%', transform: 'translateY(-50%)',
                                    background: 'none', border: 'none', padding: 4, color: 'var(--text-muted)',
                                    cursor: 'pointer'
                                }}
                            >
                                {showPw ? <EyeOff size={16} /> : <Eye size={16} />}
                            </button>
                        </div>
                    </div>

                    {error && <div className="login-error">{error}</div>}

                    <button type="submit" className="btn-primary" disabled={loading}
                        style={{ width: '100%', justifyContent: 'center', marginTop: 8 }}>
                        <LogIn size={16} />
                        {loading ? 'Signing in...' : 'Sign In'}
                    </button>
                </form>

                <button
                    className="btn btn-sm"
                    onClick={toggle}
                    style={{ width: '100%', justifyContent: 'center', marginTop: 16 }}
                >
                    {isDark ? <Sun size={14} /> : <Moon size={14} />}
                    {isDark ? 'Light Mode' : 'Dark Mode'}
                </button>
            </div>
        </div>
    )
}
