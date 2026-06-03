import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './context/AuthContext'
import Layout from './components/Layout'
import Login from './pages/Login'
import Setup from './pages/Setup'
import Dashboard from './pages/Dashboard'
import Tenants from './pages/Tenants'
import Branches from './pages/Branches'
import Users from './pages/Users'
import Invoices from './pages/Invoices'
import InvoiceDetail from './pages/InvoiceDetail'
import HitlQueue from './pages/HitlQueue'
import Rules from './pages/Rules'
import Conversations from './pages/Conversations'
import StorageConfig from './pages/StorageConfig'
import Archive from './pages/Archive'
import AuditLogs from './pages/AuditLogs'
import Settings from './pages/Settings'

function RequireAuth({ children }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="min-h-screen flex items-center justify-center text-gray-400">กำลังโหลด…</div>
  if (!user) return <Navigate to="/login" replace />
  return children
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/setup" element={<Setup />} />
          <Route path="/" element={<RequireAuth><Layout /></RequireAuth>}>
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="tenants" element={<Tenants />} />
            <Route path="branches" element={<Branches />} />
            <Route path="users" element={<Users />} />
            <Route path="invoices" element={<Invoices />} />
            <Route path="invoices/:id" element={<InvoiceDetail />} />
            <Route path="hitl" element={<HitlQueue />} />
            <Route path="rules" element={<Rules />} />
            <Route path="conversations" element={<Conversations />} />
            <Route path="storage" element={<StorageConfig />} />
            <Route path="archive" element={<Archive />} />
            <Route path="audit-logs" element={<AuditLogs />} />
            <Route path="settings" element={<Settings />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
