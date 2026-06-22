import { useEffect, useRef, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Input, StatusBadge, useForm } from '../components/ui'

const INIT = { name: '', tax_id: '', address: '', business_type: 'service', status: 'active' }

const BIZ_LABELS = { trading: 'ซื้อมาขายไป', service: 'บริการ', construction: 'รับเหมาก่อสร้าง' }

export default function Tenants() {
  const [data, setData]         = useState([])
  const [modal, setModal]       = useState(null)
  const [selected, setSelected] = useState(null)
  const [form, onChange, reset, setForm] = useForm(INIT)
  const [error, setError]       = useState('')
  const [ocrLoading, setOcrLoading] = useState(false)
  const [extractedBranches, setExtractedBranches] = useState([])
  const fileRef = useRef()

  const load = () => api.get('/tenants').then((r) => setData(r.data.data ?? []))
  useEffect(() => { load() }, [])

  const openCreate = () => { reset(); setError(''); setExtractedBranches([]); setModal('create') }

  const handleOCR = async (e) => {
    const file = e.target.files?.[0]
    if (!file) return
    setOcrLoading(true); setError('')
    try {
      const fd = new FormData()
      fd.append('file', file)
      const res = await api.post('/ocr/extract-company', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
      const d = res.data.data
      setForm((f) => ({
        ...f,
        name:          d.name          || f.name,
        tax_id:        d.tax_id        || f.tax_id,
        address:       d.address       || f.address,
        business_type: d.business_type || f.business_type,
      }))
      setExtractedBranches(d.branches ?? [])
    } catch (err) {
      setError('OCR ไม่สำเร็จ: ' + (err.response?.data?.error || err.message))
    } finally {
      setOcrLoading(false)
      e.target.value = ''
    }
  }
  const openEdit   = (row) => {
    setForm({ name: row.name, tax_id: row.tax_id, address: row.address || '', business_type: row.business_type || 'service', status: row.status })
    setSelected(row); setError(''); setModal('edit')
  }

  const submit = async (e) => {
    e.preventDefault(); setError('')
    try {
      if (modal === 'create') {
        const res = await api.post('/tenants', { name: form.name, tax_id: form.tax_id, business_type: form.business_type, address: form.address, status: form.status })
        const tenantId = res.data.data?.id
        if (tenantId && extractedBranches.length > 0) {
          await Promise.all(extractedBranches.map((b) =>
            api.post(`/tenants/${tenantId}/branches`, { name: b.name || 'สาขา', code: b.code || '00001', address: b.address, phone: b.phone, status: 'active' })
              .catch(() => {}) // ไม่ block ถ้า branch สร้างไม่ได้
          ))
        }
      } else {
        await api.put(`/tenants/${selected.id}`, {
          name: form.name, address: form.address, status: form.status, business_type: form.business_type,
        })
      }
      setModal(null); load()
    } catch (err) { setError(err.message) }
  }

  const cols = [
    { key: 'id',      label: 'ID',      render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'name',    label: 'ชื่อบริษัท' },
    { key: 'tax_id',  label: 'เลขผู้เสียภาษี', render: (r) => <span className="font-mono text-sm">{r.tax_id}</span> },
    { key: 'business_type', label: 'ประเภทธุรกิจ', render: (r) => BIZ_LABELS[r.business_type] || r.business_type || '—' },
    { key: 'address', label: 'ที่อยู่', render: (r) => r.address
        ? <span className="text-xs text-gray-600 truncate max-w-xs block" title={r.address}>{r.address}</span>
        : <span className="text-xs text-amber-500">ยังไม่ได้กรอก</span>
    },
    { key: 'status',  label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
  ]

  const taCls = 'w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-400 resize-y'

  return (
    <div>
      <PageHeader title="Tenant Management" action={<Btn onClick={openCreate}>+ เพิ่ม Tenant</Btn>} />
      <Table columns={cols} data={data} onRowClick={openEdit} />

      {modal && (
        <Modal title={modal === 'create' ? 'เพิ่ม Tenant' : 'แก้ไข Tenant'} devLabel={modal === 'create' ? 'P-01-M Create' : 'P-01-M Edit'} onClose={() => setModal(null)}>
          <form onSubmit={submit} className="space-y-3">
            {/* OCR auto-fill — create mode only */}
            {modal === 'create' && (
              <div className="flex items-center gap-2 pb-1 border-b border-gray-100">
                <input ref={fileRef} type="file" accept="image/jpeg,image/png" className="hidden" onChange={handleOCR} />
                <button type="button" onClick={() => fileRef.current?.click()}
                  disabled={ocrLoading}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded border border-indigo-300 text-indigo-700 bg-indigo-50 hover:bg-indigo-100 disabled:opacity-50 transition-colors">
                  {ocrLoading ? '⏳ กำลังอ่าน…' : '📷 อ่านเอกสาร (auto-fill)'}
                </button>
                <span className="text-xs text-gray-400">รองรับรูปภาพ JPG/PNG เท่านั้น (PDF ยังไม่รองรับ)</span>
              </div>
            )}
            {/* ID — read-only เฉพาะ edit, ไม่โชว์ตอน create */}
            {modal === 'edit' && (
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">ID</label>
                <p className="font-mono text-xs text-gray-400 bg-gray-50 rounded px-3 py-2 border border-gray-200 break-all">{selected?.id}</p>
              </div>
            )}
            {/* เลขผู้เสียภาษี — editable ตอน create, read-only ตอน edit */}
            {modal === 'create'
              ? <Input label="เลขผู้เสียภาษี (13 หลัก)" name="tax_id" value={form.tax_id} onChange={onChange} required />
              : <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">เลขผู้เสียภาษี</label>
                  <p className="font-mono text-sm text-gray-700 bg-gray-50 rounded px-3 py-2 border border-gray-200">{selected?.tax_id}</p>
                </div>
            }
            <Input label="ชื่อบริษัท" name="name" value={form.name} onChange={onChange} required />
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">ประเภทธุรกิจ</label>
              <select name="business_type" value={form.business_type} onChange={onChange}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
                <option value="service">บริการ</option>
                <option value="trading">ซื้อมาขายไป / ผลิต</option>
                <option value="construction">รับเหมาก่อสร้าง</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                ที่อยู่จดทะเบียน <span className="text-xs font-normal text-gray-400">(ใช้ใน header รายงานภาษีซื้อ, 50 ทวิ)</span>
              </label>
              <textarea name="address" value={form.address} onChange={onChange} rows={3} className={taCls}
                placeholder="เลขที่ ถนน แขวง/ตำบล เขต/อำเภอ จังหวัด รหัสไปรษณีย์" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
              <select name="status" value={form.status} onChange={onChange}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
                <option value="active">active</option>
                <option value="inactive">inactive</option>
              </select>
            </div>
            {/* Extracted branches preview */}
            {extractedBranches.length > 0 && (
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                <p className="text-xs font-semibold text-blue-700 mb-2">🏢 พบ {extractedBranches.length} สาขาจากเอกสาร — จะสร้างอัตโนมัติหลัง Tenant ถูกบันทึก</p>
                <ul className="space-y-1">
                  {extractedBranches.map((b, i) => (
                    <li key={i} className="text-xs text-blue-800">
                      <span className="font-mono bg-blue-100 px-1 rounded">{b.code || '—'}</span>{' '}
                      {b.name}{b.address ? ` — ${b.address.slice(0, 40)}…` : ''}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {error && <p className="text-red-500 text-sm">{error}</p>}
            <div className="flex justify-end gap-2 pt-1">
              <Btn variant="secondary" onClick={() => setModal(null)}>ยกเลิก</Btn>
              <Btn type="submit">บันทึก</Btn>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
