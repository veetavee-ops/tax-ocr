import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Input, Select, StatusBadge, useForm } from '../components/ui'

const INIT = { tenant_id: '', name: '', email: '', phone: '', line_user_id: '', role: 'staff' }

export default function Users() {
  const [data, setData]         = useState([])
  const [tenants, setTenants]   = useState([])
  const [modal, setModal]       = useState(null)
  const [selected, setSelected] = useState(null)
  const [form, onChange, reset, setForm] = useForm(INIT)
  const [error, setError]       = useState('')

  const load = () =>
    Promise.all([api.get('/users'), api.get('/tenants')]).then(([u, t]) => {
      setData(u.data.data ?? [])
      setTenants(t.data.data ?? [])
    })
  useEffect(() => { load() }, [])

  const openCreate = () => { reset(); setError(''); setModal('create') }
  const openEdit = (row) => {
    setForm({ tenant_id: row.tenant_id, name: row.name, email: row.email, phone: row.phone, line_user_id: row.line_user_id, role: row.role, status: row.status })
    setSelected(row); setError(''); setModal('edit')
  }

  const submit = async (e) => {
    e.preventDefault(); setError('')
    try {
      if (modal === 'create') {
        await api.post('/users', form)
      } else {
        await api.put(`/users/${selected.id}`, { name: form.name, email: form.email, phone: form.phone, line_user_id: form.line_user_id, role: form.role, status: form.status })
      }
      setModal(null); load()
    } catch (err) { setError(err.message) }
  }

  const deleteUser = async (row) => {
    if (!confirm(`ลบ user "${row.name}"?`)) return
    await api.delete(`/users/${row.id}`)
    load()
  }

  const tenantMap  = Object.fromEntries(tenants.map((t) => [t.id, t.name]))
  const tenantOpts = tenants.map((t) => ({ value: t.id, label: t.name }))

  const cols = [
    { key: 'id',    label: 'ID', render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'name',  label: 'ชื่อ' },
    { key: 'tenant_id', label: 'Tenant', render: (r) => tenantMap[r.tenant_id] ?? r.tenant_id },
    { key: 'email', label: 'Email' },
    { key: 'role',  label: 'Role' },
    { key: 'status',label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
    { key: 'actions', label: '', render: (r) => (
      <button onClick={(e) => { e.stopPropagation(); deleteUser(r) }}
        className="text-red-500 hover:underline text-xs">ลบ</button>
    )},
  ]

  return (
    <div>
      <PageHeader title="User Management" action={<Btn onClick={openCreate}>+ เพิ่ม User</Btn>} />
      <Table columns={cols} data={data} onRowClick={openEdit} />

      {modal && (
        <Modal title={modal === 'create' ? 'เพิ่ม User' : 'แก้ไข User'} onClose={() => setModal(null)}>
          <form onSubmit={submit}>
            {modal === 'create' && <Select label="Tenant" name="tenant_id" value={form.tenant_id} onChange={onChange} options={tenantOpts} required />}
            <Input label="ชื่อ-นามสกุล" name="name" value={form.name} onChange={onChange} required />
            <Input label="Email" name="email" value={form.email ?? ''} onChange={onChange} type="email" />
            <Input label="Phone" name="phone" value={form.phone ?? ''} onChange={onChange} />
            <Input label="LINE User ID" name="line_user_id" value={form.line_user_id ?? ''} onChange={onChange} />
            <Select label="Role" name="role" value={form.role} onChange={onChange} required
              options={[{ value: 'admin', label: 'Admin' }, { value: 'staff', label: 'Staff' }]} />
            {modal === 'edit' && (
              <Select label="Status" name="status" value={form.status} onChange={onChange}
                options={[{ value: 'active', label: 'Active' }, { value: 'inactive', label: 'Inactive' }]} />
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
