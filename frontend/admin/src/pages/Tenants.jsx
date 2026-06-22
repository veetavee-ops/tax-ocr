import { useEffect, useState } from 'react'
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

  const load = () => api.get('/tenants').then((r) => setData(r.data.data ?? []))
  useEffect(() => { load() }, [])

  const openCreate = () => { reset(); setError(''); setModal('create') }
  const openEdit   = (row) => {
    setForm({ name: row.name, tax_id: row.tax_id, address: row.address || '', business_type: row.business_type || 'service', status: row.status })
    setSelected(row); setError(''); setModal('edit')
  }

  const submit = async (e) => {
    e.preventDefault(); setError('')
    try {
      if (modal === 'create') {
        await api.post('/tenants', { name: form.name, tax_id: form.tax_id, business_type: form.business_type })
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
            <Input label="ชื่อบริษัท" name="name" value={form.name} onChange={onChange} required />
            {modal === 'create' && (
              <Input label="เลขผู้เสียภาษี (13 หลัก)" name="tax_id" value={form.tax_id} onChange={onChange} required />
            )}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">ประเภทธุรกิจ</label>
              <select name="business_type" value={form.business_type} onChange={onChange}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
                <option value="service">บริการ</option>
                <option value="trading">ซื้อมาขายไป / ผลิต</option>
                <option value="construction">รับเหมาก่อสร้าง</option>
              </select>
            </div>
            {modal === 'edit' && (
              <>
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
              </>
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
