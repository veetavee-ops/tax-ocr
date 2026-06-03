import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'
import { useAuth } from '../context/AuthContext'

export default function Setup() {
  const navigate = useNavigate()
  const { login } = useAuth()
  const [form, setForm] = useState({ tenant_name: '', tax_id: '', name: '', email: '', password: '' })
  const [error, setError]   = useState('')
  const [loading, setLoading] = useState(false)

  const onChange = (e) => setForm((f) => ({ ...f, [e.target.name]: e.target.value }))

  const submit = async (e) => {
    e.preventDefault()
    setError(''); setLoading(true)
    try {
      await api.post('/auth/setup', form)
      await login(form.email, form.password)
      navigate('/dashboard')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const fields = [
    { name: 'tenant_name', label: 'ชื่อบริษัท' },
    { name: 'tax_id',      label: 'เลขผู้เสียภาษี 13 หลัก' },
    { name: 'name',        label: 'ชื่อ Admin' },
    { name: 'email',       label: 'Email', type: 'email' },
    { name: 'password',    label: 'Password', type: 'password' },
  ]

  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center">
      <div className="bg-white rounded-lg shadow-md p-8 w-full max-w-sm">
        <h1 className="text-2xl font-bold text-gray-800 mb-1">ตั้งค่าระบบ</h1>
        <p className="text-sm text-gray-500 mb-6">สร้างบัญชี Admin และบริษัทแรก</p>
        <form onSubmit={submit}>
          {fields.map((f) => (
            <div key={f.name} className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-1">{f.label}</label>
              <input
                type={f.type || 'text'} name={f.name} value={form[f.name]}
                onChange={onChange} required
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          ))}
          {error && <p className="text-red-500 text-sm mb-4">{error}</p>}
          <button type="submit" disabled={loading}
            className="w-full bg-blue-600 text-white py-2 rounded font-medium hover:bg-blue-700 disabled:opacity-50">
            {loading ? 'กำลังสร้าง…' : 'สร้างระบบ'}
          </button>
        </form>
        <p className="text-center text-sm text-gray-400 mt-4">
          มีบัญชีแล้ว? <a href="/login" className="text-blue-600 hover:underline">เข้าสู่ระบบ</a>
        </p>
      </div>
    </div>
  )
}
