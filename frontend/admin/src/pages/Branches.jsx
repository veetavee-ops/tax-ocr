import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Input, Select, StatusBadge, useForm } from '../components/ui'

const INIT = { tenant_id: '', name: '', code: '' }

export default function Branches() {
  const [data, setData]         = useState([])
  const [tenants, setTenants]   = useState([])
  const [modal, setModal]       = useState(null)
  const [selected, setSelected] = useState(null)
  const [form, onChange, reset, setForm] = useForm(INIT)
  const [error, setError]       = useState('')

  const load = async () => {
    const [td, bd] = await Promise.allSettled([api.get('/tenants'), api.get('/tenants')])
    const ts = td.value?.data?.data ?? []
    setTenants(ts)
    // load branches for all tenants
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
    setForm({ tenant_id: row.tenant_id, name: row.name, code: row.code, status: row.status })
    setSelected(row); setError(''); setModal('edit')
  }

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      if (modal === 'create') {
        await api.post(`/tenants/${form.tenant_id}/branches`, { name: form.name, code: form.code })
      } else {
        await api.put(`/tenants/${selected.tenant_id}/branches/${selected.id}`, { name: form.name, status: form.status })
      }
      setModal(null); load()
    } catch (err) { setError(err.message) }
  }

  const cols = [
    { key: 'id',    label: 'ID', render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'tenant_name', label: 'Tenant' },
    { key: 'name',  label: 'ชื่อสาขา' },
    { key: 'code',  label: 'รหัส' },
    { key: 'status',label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
  ]

  const tenantOpts = tenants.map((t) => ({ value: t.id, label: t.name }))

  return (
    <div>
      <PageHeader title="Branch Management" action={<Btn onClick={openCreate}>+ เพิ่ม Branch</Btn>} />
      <Table columns={cols} data={data} onRowClick={openEdit} />

      {modal && (
        <Modal title={modal === 'create' ? 'เพิ่ม Branch' : 'แก้ไข Branch'} onClose={() => setModal(null)}>
          <form onSubmit={submit}>
            {modal === 'create' && <Select label="Tenant" name="tenant_id" value={form.tenant_id} onChange={onChange} options={tenantOpts} required />}
            <Input label="ชื่อสาขา" name="name" value={form.name} onChange={onChange} required />
            {modal === 'create' && <Input label="รหัสสาขา" name="code" value={form.code} onChange={onChange} required />}
            {modal === 'edit' && (
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                <select name="status" value={form.status} onChange={onChange}
                  className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
                  <option value="active">active</option>
                  <option value="inactive">inactive</option>
                </select>
              </div>
            )}
            {error && <p className="text-red-500 text-sm mb-3">{error}</p>}
            <div className="flex justify-end gap-2">
              <Btn variant="secondary" onClick={() => setModal(null)}>ยกเลิก</Btn>
              <Btn type="submit">บันทึก</Btn>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
