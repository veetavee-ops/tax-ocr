import React, { useEffect, useRef, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Input, StatusBadge, useForm } from '../components/ui'

const INIT = { name: '', tax_id: '', address: '', business_type: 'service', status: 'active' }
const BIZ_LABELS = { trading: 'ซื้อมาขายไป', service: 'บริการ', construction: 'รับเหมาก่อสร้าง' }

function useDblClickProtect(isFocused) {
  const ref = useRef(false)
  const onMouseDown = () => { ref.current = !isFocused }
  const guard = (fn) => () => { if (ref.current) { ref.current = false; return } fn() }
  return { onMouseDown, guard }
}

function ConfirmDialog({ title, message, warning, confirmLabel, confirmClass, onConfirm, onCancel }) {
  const [focused, setFocused] = useState('cancel')
  const cancelProtect  = useDblClickProtect(focused === 'cancel')
  const confirmProtect = useDblClickProtect(focused === 'confirm')

  return (
    <Modal title={title} onClose={onCancel} hideClose>
      <p className="text-sm text-gray-700 mb-1">{message}</p>
      {warning && <p className="text-xs text-red-500 mb-4">{warning}</p>}
      <div className="flex justify-end gap-3 mt-4">
        <button autoFocus
          onFocus={() => setFocused('cancel')}
          onMouseDown={cancelProtect.onMouseDown}
          onClick={cancelProtect.guard(onCancel)}
          className={`px-6 py-2 rounded-lg font-semibold text-sm transition-all focus:outline-none ${
            focused === 'cancel' ? 'bg-blue-500 text-white scale-105' : 'bg-gray-200 text-gray-400'}`}>
          ยกเลิก
        </button>
        <button
          onFocus={() => setFocused('confirm')}
          onMouseDown={confirmProtect.onMouseDown}
          onClick={confirmProtect.guard(onConfirm)}
          className={`px-6 py-2 rounded-lg font-semibold text-sm transition-all focus:outline-none ${
            focused === 'confirm' ? `${confirmClass} scale-105` : 'bg-red-100 text-red-400'}`}>
          {confirmLabel}
        </button>
      </div>
    </Modal>
  )
}


export default function Tenants() {
  const [tab, setTab]           = useState('active')
  const [data, setData]         = useState([])
  const [trash, setTrash]       = useState([])
  const [modal, setModal]       = useState(null)
  const [selected, setSelected] = useState(null)
  const [form, onChange, reset, setForm] = useForm(INIT)
  const [error, setError]       = useState('')
  const [ocrLoading, setOcrLoading] = useState(false)
  const [extractedBranches, setExtractedBranches] = useState([])
  const fileRef = useRef()

  // confirm dialogs
  const [confirmTrash, setConfirmTrash]       = useState(null)
  const [confirmPermanent, setConfirmPermanent] = useState(null)
  const [confirmRestore, setConfirmRestore]   = useState(null)

  const [confirmSuspend, setConfirmSuspend]     = useState(null)
  const [confirmUnsuspend, setConfirmUnsuspend] = useState(null)

  const loadActive = () => api.get('/tenants').then((r) => setData(r.data.data ?? []))
  const loadTrash  = () => api.get('/tenants/trash').then((r) => setTrash(r.data.data ?? []))
  const loadAll    = () => { loadActive(); loadTrash() }

  useEffect(() => { loadAll() }, [])

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

  const openEdit = (row) => {
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
              .catch(() => {})
          ))
        }
      } else {
        await api.put(`/tenants/${selected.id}`, {
          name: form.name, address: form.address, status: form.status, business_type: form.business_type,
        })
      }
      setModal(null); loadAll()
    } catch (err) { setError(err.message) }
  }

  // trash actions
  const doTrash = async () => {
    await api.delete(`/tenants/${confirmTrash.id}`)
    setConfirmTrash(null); loadAll()
  }
  const doPermanent = async () => {
    await api.delete(`/tenants/${confirmPermanent.id}/permanent`)
    setConfirmPermanent(null); loadTrash()
  }
  const doRestore = async () => {
    await api.post(`/tenants/${confirmRestore.id}/restore`)
    setConfirmRestore(null); loadAll()
  }

  const doSuspend = async () => {
    await api.post(`/tenants/${confirmSuspend.id}/suspend`, { reason: '' })
    setConfirmSuspend(null); loadActive()
  }
  const doUnsuspend = async () => {
    await api.post(`/tenants/${confirmUnsuspend.id}/unsuspend`)
    setConfirmUnsuspend(null); loadActive()
  }

  const activeCols = [
    { key: 'id',     label: 'ID',     render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'name',   label: 'ชื่อบริษัท' },
    { key: 'tax_id', label: 'เลขผู้เสียภาษี', render: (r) => <span className="font-mono text-sm">{r.tax_id}</span> },
    { key: 'business_type', label: 'ประเภทธุรกิจ', render: (r) => BIZ_LABELS[r.business_type] || r.business_type || '—' },
    { key: 'address', label: 'ที่อยู่', render: (r) => r.address
        ? <span className="text-xs text-gray-600 truncate max-w-xs block" title={r.address}>{r.address}</span>
        : <span className="text-xs text-amber-500">ยังไม่ได้กรอก</span>
    },
    { key: 'status', label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
    { key: '_actions', label: '', render: (r) => (
      <div className="flex -my-3 gap-0" onClick={(e) => e.stopPropagation()}>
        {r.status === 'suspended'
          ? <button onClick={() => setConfirmUnsuspend(r)}
              className="text-xs text-green-600 hover:text-green-800 px-3 py-3 hover:bg-green-50 whitespace-nowrap">
              เปิดบริการ
            </button>
          : r.status === 'active' &&
            <button onClick={() => setConfirmSuspend(r)}
              className="text-xs text-red-500 hover:text-red-700 px-3 py-3 hover:bg-red-50 whitespace-nowrap">
              ปิดบริการ
            </button>
        }
        <button onClick={() => setConfirmTrash(r)}
          className="text-xs text-red-500 hover:text-red-700 px-3 py-3 hover:bg-red-50">
          ลบ
        </button>
      </div>
    )},
  ]

  const trashCols = [
    { key: 'id',     label: 'ID',     render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'name',   label: 'ชื่อบริษัท', render: (r) => <span className="line-through text-gray-400">{r.name}</span> },
    { key: 'tax_id', label: 'เลขผู้เสียภาษี', render: (r) => <span className="font-mono text-sm text-gray-400">{r.tax_id}</span> },
    { key: 'deleted_at', label: 'ลบเมื่อ', render: (r) => r.deleted_at ? new Date(r.deleted_at).toLocaleString('th-TH') : '—' },
    { key: '_actions', label: '', render: (r) => (
      <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
        <button onClick={() => setConfirmRestore(r)}
          className="text-xs text-blue-500 hover:text-blue-700 px-2 py-1 rounded hover:bg-blue-50">
          เรียกคืน
        </button>
        <button onClick={() => setConfirmPermanent(r)}
          className="text-xs text-red-500 hover:text-red-700 px-2 py-1 rounded hover:bg-red-50 whitespace-nowrap">
          ลบถาวร
        </button>
      </div>
    )},
  ]

  const taCls = 'w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-400 resize-y'

  return (
    <div>
      <PageHeader title="Tenant Management" action={tab === 'active' && <Btn onClick={openCreate}>+ เพิ่ม Tenant</Btn>} />

      {/* Tabs */}
      <div className="flex gap-1 mb-4 border-b border-gray-200">
        <button onClick={() => setTab('active')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${tab === 'active' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'}`}>
          ใช้งานอยู่ ({data.length})
        </button>
        <button onClick={() => setTab('trash')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${tab === 'trash' ? 'border-red-500 text-red-600' : 'border-transparent text-gray-500 hover:text-gray-700'}`}>
          ถังขยะ {trash.length > 0 && <span className="ml-1 bg-red-100 text-red-600 text-xs px-1.5 py-0.5 rounded-lg">{trash.length}</span>}
        </button>
      </div>

      {tab === 'active'
        ? <Table columns={activeCols} data={data} onRowClick={openEdit} />
        : <Table columns={trashCols} data={trash} />
      }

      {/* Confirm: ย้ายไปถังขยะ */}
      {confirmTrash && (
        <ConfirmDialog
          title="ย้ายไปถังขยะ"
          message={<>ย้าย <span className="font-semibold">{confirmTrash.name}</span> ไปถังขยะ?</>}
          warning="สามารถเรียกคืนได้ภายหลัง"
          confirmLabel="ย้ายไปถังขยะ"
          confirmClass="bg-red-500 text-white"
          onConfirm={doTrash}
          onCancel={() => setConfirmTrash(null)}
        />
      )}

      {/* Confirm: ลบถาวร */}
      {confirmPermanent && (
        <ConfirmDialog
          title="ลบถาวร"
          message={<>ลบ <span className="font-semibold">{confirmPermanent.name}</span> ออกจากระบบถาวร?</>}
          warning="ข้อมูลทั้งหมดของ Tenant นี้จะหายไปและไม่สามารถเรียกคืนได้"
          confirmLabel="ลบถาวร"
          confirmClass="bg-red-600 text-white"
          onConfirm={doPermanent}
          onCancel={() => setConfirmPermanent(null)}
        />
      )}

      {/* Confirm: เรียกคืน */}
      {confirmRestore && (
        <ConfirmDialog
          title="เรียกคืน Tenant"
          message={<>เรียกคืน <span className="font-semibold">{confirmRestore.name}</span> กลับมาใช้งาน?</>}
          warning={null}
          confirmLabel="เรียกคืน"
          confirmClass="bg-green-500 text-white"
          onConfirm={doRestore}
          onCancel={() => setConfirmRestore(null)}
        />
      )}

      {/* Confirm: ปิดบริการ */}
      {confirmSuspend && (
        <ConfirmDialog
          title="ปิดการให้บริการ"
          message={<>ปิดบริการ <span className="font-semibold">{confirmSuspend.name}</span>?</>}
          warning="Tenant จะไม่สามารถใช้งานระบบได้จนกว่าจะเปิดบริการใหม่"
          confirmLabel="ปิดบริการ"
          confirmClass="bg-red-500 text-white"
          onConfirm={doSuspend}
          onCancel={() => setConfirmSuspend(null)}
        />
      )}

      {/* Confirm: เปิดบริการ (Unsuspend) */}
      {confirmUnsuspend && (
        <ConfirmDialog
          title="เปิดการให้บริการ"
          message={<>เปิดบริการ <span className="font-semibold">{confirmUnsuspend.name}</span> อีกครั้ง?</>}
          warning={null}
          confirmLabel="เปิดบริการ"
          confirmClass="bg-green-500 text-white"
          onConfirm={doUnsuspend}
          onCancel={() => setConfirmUnsuspend(null)}
        />
      )}

      {/* Modal: Create / Edit */}
      {modal && (
        <Modal title={modal === 'create' ? 'เพิ่ม Tenant' : 'แก้ไข Tenant'} devLabel={modal === 'create' ? 'P-01-M Create' : 'P-01-M Edit'} onClose={() => setModal(null)}>
          <form onSubmit={submit} className="space-y-3">
            {modal === 'create' && (
              <div className="flex items-center gap-2 pb-1 border-b border-gray-100">
                <input ref={fileRef} type="file" accept="image/jpeg,image/png,application/pdf" className="hidden" onChange={handleOCR} />
                <button type="button" onClick={() => fileRef.current?.click()} disabled={ocrLoading}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded border border-indigo-300 text-indigo-700 bg-indigo-50 hover:bg-indigo-100 disabled:opacity-50 transition-colors">
                  {ocrLoading ? '⏳ กำลังอ่าน…' : '📷 อ่านเอกสาร (JPG/PNG/PDF)'}
                </button>
                <span className="text-xs text-gray-400">รองรับรูปภาพ JPG/PNG เท่านั้น</span>
              </div>
            )}
            {modal === 'edit' && (
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">ID</label>
                <p className="font-mono text-xs text-gray-400 bg-gray-50 rounded px-3 py-2 border border-gray-200 break-all">{selected?.id}</p>
              </div>
            )}
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
                ที่อยู่จดทะเบียน <span className="text-xs font-normal text-gray-400">(ใช้ใน header รายงานภาษีซื้อ)</span>
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
