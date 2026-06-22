import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Input, Select, StatusBadge, useForm } from '../components/ui'

const INIT = { tenant_id: '', name: '', code: '', address: '', phone: '', status: 'active' }

export default function Branches() {
  const [data, setData]         = useState([])
  const [tenants, setTenants]   = useState([])
  const [modal, setModal]       = useState(null)
  const [selected, setSelected] = useState(null)
  const [form, onChange, reset, setForm] = useForm(INIT)
  const [error, setError]       = useState('')

  const load = async () => {
    const ts = (await api.get('/tenants').catch(() => ({ data: { data: [] } }))).data?.data ?? []
    setTenants(ts)
    const allBranches = (
      await Promise.all(ts.map((t) => api.get(`/tenants/${t.id}/branches`).catch(() => ({ data: { data: [] } }))))
    ).flatMap((r, i) =>
      (r.data?.data ?? []).map((b) => ({ ...b, tenant_name: ts[i]?.name }))
    )
    setData(allBranches)
  }
  useEffect(() => { load() }, [])

  const openCreate = () => { reset(); setError(''); setModal('create') }
  const openEdit   = (row) => {
    setForm({ tenant_id: row.tenant_id, name: row.name, code: row.code,
              address: row.address || '', phone: row.phone || '', status: row.status })
    setSelected(row); setError(''); setModal('edit')
  }

  const submit = async (e) => {
    e.preventDefault(); setError('')
    try {
      if (modal === 'create') {
        await api.post(`/tenants/${form.tenant_id}/branches`, {
          name: form.name, code: form.code, address: form.address, phone: form.phone,
        })
      } else {
        await api.put(`/tenants/${selected.tenant_id}/branches/${selected.id}`, {
          name: form.name, address: form.address, phone: form.phone, status: form.status,
        })
      }
      setModal(null); load()
    } catch (err) { setError(err.message) }
  }

  const cols = [
    { key: 'id',          label: 'ID',       render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'tenant_name', label: 'Tenant' },
    { key: 'name',        label: 'ชื่อสาขา' },
    { key: 'code',        label: 'รหัส', render: (r) => <span className="font-mono text-sm">{r.code || '—'}</span> },
    { key: 'phone',       label: 'เบอร์โทร', render: (r) => r.phone || <span className="text-gray-300">—</span> },
    { key: 'address',     label: 'ที่อยู่', render: (r) => r.address
        ? <span className="text-xs text-gray-600 truncate max-w-xs block" title={r.address}>{r.address}</span>
        : <span className="text-xs text-amber-500">ยังไม่ได้กรอก</span>
    },
    { key: 'status',      label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
  ]

  const tenantOpts = tenants.map((t) => ({ value: t.id, label: t.name }))
  const taCls = 'w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-400 resize-y'

  return (
    <div>
      <PageHeader title="Branch Management" action={<Btn onClick={openCreate}>+ เพิ่ม Branch</Btn>} />
      <Table columns={cols} data={data} onRowClick={openEdit} />

      {modal && (
        <Modal title={modal === 'create' ? 'เพิ่ม Branch' : 'แก้ไข Branch'} onClose={() => setModal(null)}>
          <form onSubmit={submit} className="space-y-3">
            {modal === 'create' && (
              <Select label="Tenant" name="tenant_id" value={form.tenant_id} onChange={onChange} options={tenantOpts} required />
            )}
            <Input label="ชื่อสาขา" name="name" value={form.name} onChange={onChange} required />
            {modal === 'create' && (
              <Input label="รหัสสาขา (5 หลัก, 00000 = สำนักงานใหญ่)" name="code" value={form.code} onChange={onChange} />
            )}
            <Input label="เบอร์โทรสาขา" name="phone" value={form.phone} onChange={onChange} />
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                ที่อยู่สาขา <span className="text-xs font-normal text-gray-400">(ใช้ใน header รายงานภาษีซื้อ)</span>
              </label>
              <textarea name="address" value={form.address} onChange={onChange} rows={3} className={taCls}
                placeholder="เลขที่ ถนน แขวง/ตำบล เขต/อำเภอ จังหวัด รหัสไปรษณีย์" />
            </div>
            {modal === 'edit' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                <select name="status" value={form.status} onChange={onChange}
                  className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
                  <option value="active">active</option>
                  <option value="inactive">inactive</option>
                </select>
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
