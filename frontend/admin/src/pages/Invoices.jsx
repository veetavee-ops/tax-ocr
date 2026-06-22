import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'
import { useAuth } from '../context/AuthContext'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Select, StatusBadge } from '../components/ui'

// Thai month names (CE)
const MONTH_NAMES = ['', 'ม.ค.', 'ก.พ.', 'มี.ค.', 'เม.ย.', 'พ.ค.', 'มิ.ย.', 'ก.ค.', 'ส.ค.', 'ก.ย.', 'ต.ค.', 'พ.ย.', 'ธ.ค.']

function currentTHYear() {
  return new Date().getFullYear() // CE year — accounting uses CE
}

function UploadModal({ onClose }) {
  const { user } = useAuth()
  const [branches, setBranches] = useState([])
  const [branchID, setBranchID] = useState('')
  const [file, setFile] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
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
          {file && <p className="text-xs text-gray-400 mt-1">{file.name} ({(file.size / 1024).toFixed(1)} KB)</p>}
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

function PeriodModal({ invoice, onClose }) {
  const [year, setYear] = useState(invoice.accounting_year || currentTHYear())
  const [month, setMonth] = useState(invoice.accounting_month || new Date().getMonth() + 1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const yearOptions = Array.from({ length: 5 }, (_, i) => currentTHYear() - 2 + i)

  const submit = async (e) => {
    e.preventDefault()
    setLoading(true); setError('')
    try {
      await api.put(`/invoices/${invoice.id}/accounting-period`, {
        accounting_year: Number(year),
        accounting_month: Number(month),
      })
      onClose(true)
    } catch (err) {
      setError(err.response?.data?.error || err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal title="เปลี่ยนรอบบัญชีภาษี" onClose={() => onClose(false)}>
      <p className="text-sm text-gray-500 mb-4">
        เอกสาร #{invoice.invoice_no} — เลขที่ {invoice.invoice_doc_no || '–'}
      </p>
      <form onSubmit={submit}>
        <div className="flex gap-3 mb-4">
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">ปี (CE)</label>
            <select value={year} onChange={(e) => setYear(e.target.value)}
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
              {yearOptions.map((y) => <option key={y} value={y}>{y}</option>)}
            </select>
          </div>
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">เดือน</label>
            <select value={month} onChange={(e) => setMonth(e.target.value)}
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
              {MONTH_NAMES.slice(1).map((name, i) => (
                <option key={i + 1} value={i + 1}>{i + 1} — {name}</option>
              ))}
            </select>
          </div>
        </div>
        {error && <p className="text-red-500 text-sm mb-3">{error}</p>}
        <div className="flex justify-end gap-2">
          <Btn variant="secondary" onClick={() => onClose(false)}>ยกเลิก</Btn>
          <Btn type="submit" disabled={loading}>{loading ? 'กำลังบันทึก…' : 'บันทึก'}</Btn>
        </div>
      </form>
    </Modal>
  )
}

const DOC_TABS = [
  { key: '', label: 'ทั้งหมด' },
  { key: 'tax_invoice', label: 'ใบกำกับภาษี' },
  { key: 'receipt', label: 'ใบเสร็จรับเงิน' },
  { key: 'invoice_billing', label: 'ใบแจ้งหนี้' },
  { key: 'delivery_order', label: 'ใบส่งสินค้า' },
]

const DOC_TYPE_LABELS = {
  tax_invoice: 'ใบกำกับภาษี',
  receipt: 'ใบเสร็จรับเงิน',
  invoice_billing: 'ใบแจ้งหนี้',
  delivery_order: 'ใบส่งสินค้า',
}

const INVALID_REASON_LABELS = {
  buyer_tax_id_mismatch: 'เลขผู้เสียภาษีผู้ซื้อไม่ตรง',
  buyer_branch_code_mismatch: 'รหัสสาขาผู้ซื้อไม่ตรง',
  buyer_name_mismatch: 'ชื่อผู้ซื้อไม่ตรง',
  late_invoice_vat_unclaimed: 'บิลเกิน 3 เดือน',
}

export default function Invoices() {
  const { user } = useAuth()
  const now = new Date()
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(false)
  const [docTab, setDocTab] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [acctYear, setAcctYear] = useState(now.getFullYear())
  const [acctMonth, setAcctMonth] = useState(now.getMonth() + 1) // 1-based
  const [showUpload, setShowUpload] = useState(false)
  const [periodTarget, setPeriodTarget] = useState(null) // invoice being edited
  const navigate = useNavigate()

  const yearOptions = Array.from({ length: 5 }, (_, i) => now.getFullYear() - 2 + i)

  const load = () => {
    setLoading(true)
    const params = new URLSearchParams()
    if (user?.tenant_id) params.set('tenant_id', user.tenant_id)
    if (docTab) params.set('doc_type', docTab)
    if (statusFilter) params.set('status', statusFilter)
    if (acctYear) params.set('accounting_year', acctYear)
    if (acctMonth) params.set('accounting_month', acctMonth)
    api.get(`/invoices?${params}`)
      .then((r) => setData(r.data.data ?? []))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [docTab, statusFilter, acctYear, acctMonth, user?.tenant_id])

  const fmt = (n) => {
    const parts = Number(n ?? 0).toFixed(2).split('.')
    parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    return parts.join('.')
  }

  const fmtPeriod = (y, m) => {
    if (!y || !m) return '–'
    return `${MONTH_NAMES[m]} ${y}`
  }

  const handleDelete = async (e, id) => {
    e.stopPropagation()
    if (!window.confirm('ลบเอกสารนี้? ไม่สามารถกู้คืนได้')) return
    try {
      await api.delete(`/invoices/${id}`)
      load()
    } catch { /* ignore */ }
  }

  const cols = [
    {
      key: 'invoice_no', label: '#', render: (r) => (
        <div>
          <span className="font-semibold text-gray-800">#{r.invoice_no}</span>
          {r.invoice_doc_no && (
            <div className="text-xs text-gray-500 font-mono">{r.invoice_doc_no}</div>
          )}
        </div>
      )
    },
    {
      key: 'doc_type', label: 'ประเภท', render: (r) => {
        const colors = {
          tax_invoice: 'bg-blue-100 text-blue-700',
          receipt: 'bg-green-100 text-green-700',
          invoice_billing: 'bg-purple-100 text-purple-700',
          delivery_order: 'bg-gray-100 text-gray-600',
        }
        return (
          <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${colors[r.doc_type] ?? 'bg-gray-100 text-gray-600'}`}>
            {DOC_TYPE_LABELS[r.doc_type] || r.doc_type || '—'}
          </span>
        )
      }
    },
    { key: 'vendor_name', label: 'ชื่อร้าน', render: (r) => r.vendor_name || <span className="text-gray-300">—</span> },
    {
      key: 'invoice_date', label: 'วันที่ในใบ', render: (r) => (
        <div className="text-sm">
          {r.invoice_date || (r.invoice_year
            ? `${r.invoice_day || '?'}/${r.invoice_month}/${r.invoice_year}`
            : <span className="text-gray-300">—</span>
          )}
        </div>
      )
    },
    {
      key: 'accounting_period', label: 'รอบบัญชี', render: (r) => (
        <button
          onClick={(e) => { e.stopPropagation(); setPeriodTarget(r) }}
          className="text-left group"
          title="คลิกเพื่อเปลี่ยนรอบบัญชี"
        >
          <span className="text-sm font-medium text-indigo-700 group-hover:underline">
            {fmtPeriod(r.accounting_year, r.accounting_month)}
          </span>
          <span className="ml-1 text-gray-300 text-xs group-hover:text-indigo-400">✎</span>
        </button>
      )
    },
    { key: 'total_before_vat', label: 'ก่อน VAT', render: (r) => <span className="tabular-nums">{fmt(r.total_before_vat)}</span> },
    { key: 'vat_amount', label: 'VAT', render: (r) => <span className="tabular-nums">{fmt(r.vat_amount)}</span> },
    { key: 'total_amount', label: 'รวม', render: (r) => <span className="tabular-nums font-semibold">{fmt(r.total_amount)}</span> },
    {
      key: 'status', label: 'Status', render: (r) => (
        <div>
          <StatusBadge value={r.status} />
          {r.invalid_reason && r.status === 'invalid' && (
            <div className="text-xs text-red-600 mt-0.5 font-medium">
              {INVALID_REASON_LABELS[r.invalid_reason] || r.invalid_reason}
            </div>
          )}
          {r.invalid_reason === 'late_invoice_vat_unclaimed' && r.status !== 'invalid' && (
            <div className="text-xs text-amber-600 mt-0.5">⚠ บิลเกิน 3 เดือน</div>
          )}
          {r.duplicate_of && (
            <div className="text-xs text-orange-500 mt-0.5">ซ้ำ</div>
          )}
        </div>
      )
    },
    {
      key: 'actions', label: '', render: (r) => (
        <button onClick={(e) => handleDelete(e, r.id)}
          className="text-red-400 hover:text-red-600 text-xs px-2 py-1 rounded hover:bg-red-50">
          ลบ
        </button>
      )
    },
  ]

  return (
    <div>
      <PageHeader
        title="Invoice List"
        action={<Btn onClick={() => setShowUpload(true)}>+ อัปโหลดใบกำกับ</Btn>}
      />

      {/* Doc-type tabs */}
      <div className="flex gap-0 mb-4 border-b border-gray-200">
        {DOC_TABS.map((t) => (
          <button key={t.key} onClick={() => setDocTab(t.key)}
            className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${docTab === t.key
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'}`}>
            {t.label}
          </button>
        ))}
      </div>

      {/* Filters row */}
      <div className="flex flex-wrap gap-3 mb-4 items-center">
        {/* Accounting period */}
        <div className="flex items-center gap-2 bg-indigo-50 border border-indigo-200 rounded-lg px-3 py-1.5">
          <span className="text-xs font-medium text-indigo-600">รอบบัญชี</span>
          <select value={acctYear} onChange={(e) => setAcctYear(Number(e.target.value))}
            className="text-sm border-0 bg-transparent text-indigo-700 font-semibold focus:outline-none cursor-pointer">
            {yearOptions.map((y) => <option key={y} value={y}>{y}</option>)}
          </select>
          <select value={acctMonth} onChange={(e) => setAcctMonth(Number(e.target.value))}
            className="text-sm border-0 bg-transparent text-indigo-700 font-semibold focus:outline-none cursor-pointer">
            {MONTH_NAMES.slice(1).map((name, i) => (
              <option key={i + 1} value={i + 1}>{name}</option>
            ))}
          </select>
          <button onClick={() => { setAcctYear(0); setAcctMonth(0) }}
            className="text-xs text-indigo-400 hover:text-indigo-600 ml-1" title="ดูทั้งหมด">
            ✕
          </button>
        </div>

        {/* Status filter */}
        <div className="flex gap-1">
          {['', 'pending', 'verified', 'conflict', 'invalid'].map((s) => (
            <button key={s} onClick={() => setStatusFilter(s)}
              className={`px-3 py-1.5 rounded text-xs border font-medium transition-colors ${statusFilter === s
                ? 'bg-blue-600 text-white border-blue-600'
                : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'}`}>
              {s || 'ทุกสถานะ'}
            </button>
          ))}
        </div>

        {loading && <span className="text-xs text-gray-400">กำลังโหลด…</span>}
        <span className="text-xs text-gray-400 ml-auto">{data.length} รายการ</span>
      </div>

      <Table columns={cols} data={data} onRowClick={(r) => navigate(`/invoices/${r.id}`)} />

      {showUpload && (
        <UploadModal onClose={(invoiceId) => {
          setShowUpload(false)
          if (invoiceId) navigate(`/invoices/${invoiceId}`)
        }} />
      )}

      {periodTarget && (
        <PeriodModal invoice={periodTarget} onClose={(changed) => {
          setPeriodTarget(null)
          if (changed) load()
        }} />
      )}
    </div>
  )
}
