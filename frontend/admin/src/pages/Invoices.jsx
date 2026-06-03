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
      await api.post('/documents/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      onDone()
      onClose()
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

  const cols = [
    { key: 'id',     label: 'ID',   render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'vendor_tax_id', label: 'Vendor Tax ID' },
    { key: 'total_before_vat', label: 'ก่อน VAT',  render: (r) => r.total_before_vat?.toLocaleString() },
    { key: 'vat_amount',       label: 'VAT',        render: (r) => r.vat_amount?.toLocaleString() },
    { key: 'total_amount',     label: 'รวม',        render: (r) => r.total_amount?.toLocaleString() },
    { key: 'status', label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
    { key: 'created_at', label: 'วันที่', render: (r) => new Date(r.created_at).toLocaleDateString('th-TH') },
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
      {showUpload && <UploadModal onClose={() => setShowUpload(false)} onDone={load} />}
    </div>
  )
}
