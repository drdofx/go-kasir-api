import { NavLink, Outlet } from 'react-router-dom'
import { ShoppingCart, Package, Tag, BarChart3, Sun, Moon, LogOut } from 'lucide-react'
import { useTheme } from '../theme-context'
import { useAuth } from '../auth-context'

const links = [
    { to: '/app', icon: ShoppingCart, label: 'Point of Sale', end: true },
    { to: '/app/products', icon: Package, label: 'Products' },
    { to: '/app/categories', icon: Tag, label: 'Categories' },
    { to: '/app/reports', icon: BarChart3, label: 'Reports' },
]

export default function Layout() {
    const { isDark, toggle } = useTheme()
    const { user, logout } = useAuth()

    return (
        <div className="app-layout">
            <aside className="sidebar">
                <div className="sidebar-brand">
                    <h1>Kasir</h1>
                    <span>Point of Sale System</span>
                </div>
                <nav className="sidebar-nav">
                    {links.map(l => (
                        <NavLink
                            key={l.to}
                            to={l.to}
                            end={l.end}
                            className={({ isActive }) => `sidebar-link${isActive ? ' active' : ''}`}
                        >
                            <l.icon />
                            {l.label}
                        </NavLink>
                    ))}
                </nav>
                <div className="sidebar-footer">
                    {user && (
                        <div className="sidebar-user">
                            <div className="sidebar-user-avatar">{user.name.charAt(0).toUpperCase()}</div>
                            <div className="sidebar-user-info">
                                <span className="sidebar-user-name">{user.name}</span>
                                <span className="sidebar-user-role">{user.role}</span>
                            </div>
                        </div>
                    )}
                    <div className="sidebar-footer-actions">
                        <button className="btn btn-sm" onClick={toggle} title={isDark ? 'Light' : 'Dark'}>
                            {isDark ? <Sun size={14} /> : <Moon size={14} />}
                        </button>
                        <button className="btn btn-sm btn-danger" onClick={logout} title="Logout">
                            <LogOut size={14} />
                            Sign Out
                        </button>
                    </div>
                </div>
            </aside>
            <main className="main-content">
                <Outlet />
            </main>
        </div>
    )
}
