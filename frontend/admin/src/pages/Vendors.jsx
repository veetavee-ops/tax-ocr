import { useEffect, useState } from 'react'
import api from '../api/client'
import Modal from '../components/Modal'
import { PageHeader, Btn, StatusBadge } from '../components/ui'
import Table from '../components/Table'

function VendorVerifyModal({ vendor, onClose }) {
  const [form, setForm] = useState({
    name:        vendor.name || '',
    address:     vendor.address || '',
    branch_code: vendor.branch_code || '',
    branch_name: vendor.branch_name || '',
    phone:       vendor.phone || '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError]   = useState('')

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }))
  const inputCls = 'w-full border border-gray-300 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-400'

  const submit = async (e) => {
    e.preventDefault()
    if (!form.name.trim()) { setError('กรุณาระบุชื่อผู้ขาย'); return }
    setSaving(true); setError('')
    try {
      await api.post(`/vendors/${vendor.id}/verify`, form)
      onClose(true)
    } catch (err) {
      setError(err.response?.data?.error || err.message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={`ยืนยันผู้ขาย — ${vendor.tax_id}`} devLabel="P-06-M1 VendorVerify" onClose={() => onClose(false)}>
      <form onSubmit={submit} className="space-y-3 mt-1">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">ชื่อบริษัท/ร้านค้า <span className="text-red-500">*</span></label>
          <input value={form.name} onChange={set('name')} className={inputCls} />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">รหัสสาขา</label>
            <input value={form.branch_code} onChange={set('branch_code')} className={inputCls} />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">ชื่อสาขา</label>
            <input value={form.branch_name} onChange={set('branch_name')} className={inputCls} />
          </div>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">เบอร์โทร</label>
          <input value={form.phone} onChange={set('phone')} className={inputCls} />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">ที่อยู่</label>
          <textarea value={form.address} onChange={set('address')} rows={3} className={inputCls} />
        </div>
        {error && <p className="text-red-500 text-xs">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Btn variant="secondary" onClick={() => onClose(false)}>ยกเลิก</Btn>
          <Btn type="submit" disabled={saving}>{saving ? 'กำลังบันทึก…' : '✓ ยืนยันและบันทึก'}</Btn>
        </div>
      </form>
    </Modal>
  )
}

export default function Vendors() {
  const [data, setData]         = useState([])
  const [filter, setFilter]     = useState('')   // '' | 'true' | 'false'
  const [loading, setLoading]   = useState(false)
  const [target, setTarget]     = useState(null)

  const load = () => {
    setLoading(true)
    const params = filter !== '' ? `?verified=${filter}` : ''
    api.get(`/vendors${params}`)
      .then((r) => setData(r.data.data ?? []))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [filter])

  const pendingCount = data.filter((v) => !v.verified).length

  const cols = [
    {
      key: 'tax_id', label: 'เลขผู้เสียภาษี', render: (r) => (
        <span className="font-mono font-semibold text-gray-800">{r.tax_id}</span>
      )
    },
    {
      key: 'name', label: 'ชื่อ', render: (r) => (
        <span className={r.name ? 'text-gray-800' : 'text-gray-300 italic'}>
          {r.name || '— ยังไม่มีชื่อ —'}
        </span>
      )
    },
    { key: 'branch_code', label: 'สาขา', render: (r) => r.branch_code || <span className="text-gray-300">—</span> },
    {
      key: 'address', label: 'ที่อยู่', render: (r) => (
        <span className="text-xs text-gray-500 line-clamp-1">{r.address || '—'}</span>
      )
    },
    {
      key: 'verified', label: 'สถานะ', render: (r) => r.verified
        ? <span className="text-xs font-semibold text-green-600 bg-green-50 px-2 py-0.5 rounded-full">✓ ยืนยันแล้ว</span>
        : <span className="text-xs font-semibold text-amber-600 bg-amber-50 px-2 py-0.5 rounded-full">⚠ รอยืนยัน</span>
    },
    {
      key: 'actions', label: '', render: (r) => !r.verified && (
        <Btn onClick={(e) => { e.stopPropagation(); setTarget(r) }}>ยืนยัน</Btn>
      )
    },
  ]

  return (
    <div>
      <PageHeader title="ทะเบียนผู้ขาย (Vendor Registry)" />

      {pendingCount > 0 && (
        <div className="mb-4 bg-amber-50 border border-amber-300 rounded-lg px-4 py-3 flex items-center gap-3">
          <span className="text-amber-500 text-lg">⚠️</span>
          <p className="text-sm text-amber-800">
            มีผู้ขาย <strong>{pendingCount} ราย</strong> ที่ยังไม่ได้ยืนยันข้อมูล — ข้อมูลเหล่านี้อ่านมาจาก OCR อัตโนมัติ กรุณาตรวจสอบให้ถูกต้อง
          </p>
        </div>
      )}

      <div className="flex gap-2 mb-4">
        {[['', 'ทั้งหมด'], ['false', 'รอยืนยัน'], ['true', 'ยืนยันแล้ว']].map(([v, label]) => (
          <button key={v} onClick={() => setFilter(v)}
            className={`px-3 py-1.5 rounded text-sm border font-medium transition-colors ${filter === v
              ? 'bg-blue-600 text-white border-blue-600'
              : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'}`}>
            {label}
          </button>
        ))}
        {loading && <span className="text-xs text-gray-400 self-center ml-2">กำลังโหลด…</span>}
        <span className="text-xs text-gray-400 self-center ml-auto">{data.length} ราย</span>
      </div>

      <Table columns={cols} data={data} onRowClick={(r) => !r.verified && setTarget(r)} />

      {target && (
        <VendorVerifyModal vendor={target} onClose={(updated) => {
          setTarget(null)
          if (updated) load()
        }} />
      )}
    </div>
  )
}
