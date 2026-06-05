import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'
import { useAuth } from '../context/AuthContext'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Select, StatusBadge } from '../components/ui'

function UploadModal({ onClose, onDone }) {
  const { user } = useAuth()
  const [branches, setBranches] = useState([])
  const [branchID, setBranchID] = useState('')
  const [file, setFile]         = useState(null)
  const [loading, setLoading]   = useState(false)
  const [error, setError]       = useState('')
  const fileRef = useRef()

  useEffect(() => {
    if (!user?.tenant_id) return
    api.get(`/tenants/${user.tenant_id}/branches`).then((r) => {
      const list = r.data.data ?? []
      setBranches(list)
      if (list.length === 1) setBranchID(list[0].id)
    })
  }, [user])

  const submit = async (e) => {
    e.preventDefault()
    if (!file) { setError('กรุณาเลือกไฟล์'); return }
    if (!branchID) { setError('กรุณาเลือกสาขา'); return }
    setLoading(true); setError('')
    try {
      const form = new FormData()
      form.append('tenant_id', user.tenant_id)
      form.append('branch_id', branchID)
      form.append('user_id', user.id)
      form.append('file', file)
      const res = await api.post('/documents/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      onClose(res.data.invoice.id)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal title="อัปโหลดใบกำกับภาษี" onClose={onClose}>
      <form onSubmit={submit}>
        {branches.length > 1 && (
          <Select
            label="สาขา" name="branch_id" value={branchID} required
            options={branches.map((b) => ({ value: b.id, label: b.name }))}
            onChange={(e) => setBranchID(e.target.value)}
          />
        )}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            ไฟล์ <span className="text-red-500">*</span>
          </label>
          <input
            ref={fileRef} type="file" accept=".pdf,.jpg,.jpeg,.png"
            onChange={(e) => setFile(e.target.files[0])}
            className="w-full text-sm text-gray-600 border border-gray-300 rounded px-3 py-2"
          />
          {file && <p className="text-xs text-gray-400 mt-1">{file.name} ({(file.size/1024).toFixed(1)} KB)</p>}
        </div>
        {error && <p className="text-red-500 text-sm mb-3">{error}</p>}
        <div className="flex justify-end gap-2">
          <Btn variant="secondary" onClick={onClose}>ยกเลิก</Btn>
          <Btn type="submit" disabled={loading}>{loading ? 'กำลังอัปโหลด…' : 'อัปโหลด'}</Btn>
        </div>
      </form>
    </Modal>
  )
}

export default function Invoices() {
  const [data, setData]       = useState([])
  const [filter, setFilter]   = useState('')
  const [showUpload, setShowUpload] = useState(false)
  const navigate = useNavigate()

  const load = () => api.get('/invoices').then((r) => setData(r.data.data ?? []))
  useEffect(() => { load() }, [])

  const filtered = filter ? data.filter((inv) => inv.status === filter) : data

  const fmt = (n) => {
    const parts = Number(n ?? 0).toFixed(2).split('.')
    parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    return parts.join('.')
  }
  const fmtDate = (d) => new Date(d).toLocaleString('th-TH', { dateStyle: 'short', timeStyle: 'short' })

  const handleDelete = async (e, id) => {
    e.stopPropagation()
    if (!window.confirm('ลบเอกสารนี้? ไม่สามารถกู้คืนได้')) return
    try {
      await api.delete(`/invoices/${id}`)
      load()
    } catch { /* ignore */ }
  }

  const cols = [
    { key: 'invoice_no', label: '#', render: (r) => (
      <div>
        <span className="font-semibold text-gray-800">#{r.invoice_no}</span>
        <div className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</div>
      </div>
    )},
    { key: 'vendor_name', label: 'ชื่อร้าน',  render: (r) => r.vendor_name || <span className="text-gray-300">—</span> },
    { key: 'vendor_tax_id', label: 'เลขภาษี', render: (r) => r.vendor_tax_id || <span className="text-gray-300">—</span> },
    { key: 'total_before_vat', label: 'ก่อน VAT', render: (r) => fmt(r.total_before_vat) },
    { key: 'vat_amount', label: 'VAT (จากใบ)', render: (r) => fmt(r.vat_amount) },
    { key: 'total_amount',     label: 'รวม',      render: (r) => fmt(r.total_amount) },
    { key: 'status',    label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
    { key: 'created_at', label: 'วันที่-เวลา', render: (r) => fmtDate(r.created_at) },
    { key: 'actions', label: '', render: (r) => (
      <button onClick={(e) => handleDelete(e, r.id)}
        className="text-red-400 hover:text-red-600 text-xs px-2 py-1 rounded hover:bg-red-50">
        ลบ
      </button>
    )},
  ]

  return (
    <div>
      <PageHeader
        title="Invoice List"
        action={<Btn onClick={() => setShowUpload(true)}>+ อัปโหลดใบกำกับ</Btn>}
      />
      <div className="flex gap-2 mb-4">
        {['', 'pending', 'verified', 'conflict'].map((s) => (
          <button key={s} onClick={() => setFilter(s)}
            className={`px-3 py-1 rounded text-sm border ${filter === s ? 'bg-blue-600 text-white border-blue-600' : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'}`}>
            {s || 'ทั้งหมด'}
          </button>
        ))}
      </div>
      <Table columns={cols} data={filtered} onRowClick={(r) => navigate(`/invoices/${r.id}`)} />
      {showUpload && <UploadModal onClose={(invoiceId) => { setShowUpload(false); if (invoiceId) navigate(`/invoices/${invoiceId}`) }} onDone={load} />}
    </div>
  )
}
