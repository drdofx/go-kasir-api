import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ThemeProvider } from './theme-context'
import { AuthProvider, useAuth } from './auth-context'
import { CartProvider } from './cart-context'
import { ToastProvider } from './toast-context'
import Layout from './components/Layout'
import LoginPage from './pages/LoginPage'
import PosPage from './pages/PosPage'
import ProductsPage from './pages/ProductsPage'
import CategoriesPage from './pages/CategoriesPage'
import ReportsPage from './pages/ReportsPage'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth()

  if (loading) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh', color: 'var(--text-muted)' }}>
        Loading...
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/app/login" replace />
  }

  return <>{children}</>
}

function AuthRoute() {
  const { isAuthenticated, loading } = useAuth()
  if (loading) return null
  if (isAuthenticated) return <Navigate to="/app" replace />
  return <LoginPage />
}

export default function App() {
  return (
    <ThemeProvider>
      <ToastProvider>
        <CartProvider>
          <BrowserRouter>
            <AuthProvider>
              <Routes>
                <Route path="/app/login" element={<AuthRoute />} />
                <Route path="/app" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
                  <Route index element={<PosPage />} />
                  <Route path="products" element={<ProductsPage />} />
                  <Route path="categories" element={<CategoriesPage />} />
                  <Route path="reports" element={<ReportsPage />} />
                </Route>
                <Route path="*" element={<Navigate to="/app" replace />} />
              </Routes>
            </AuthProvider>
          </BrowserRouter>
        </CartProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}
