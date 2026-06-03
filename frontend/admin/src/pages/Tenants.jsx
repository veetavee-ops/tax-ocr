import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Input, StatusBadge, useForm } from '../components/ui'

const INIT = { name: '', tax_id: '' }

export default function Tenants() {
  const [data, setData]       = useState([])
  const [modal, setModal]     = useState(null) // null | 'create' | 'edit'
  const [selected, setSelected] = useState(null)
  const [form, onChange, reset, setForm] = useForm(INIT)
  const [error, setError]     = useState('')

  const load = () => api.get('/tenants').then((r) => setData(r.data.data ?? []))
  useEffect(() => { load() }, [])

  const openCreate = () => { reset(); setError(''); setModal('create') }
  const openEdit   = (row) => { setForm({ name: row.name, tax_id: row.tax_id, status: row.status }); setSelected(row); setError(''); setModal('edit') }

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      if (modal === 'create') {
        await api.post('/tenants', { name: form.name, tax_id: form.tax_id })
      } else {
        await api.put(`/tenants/${selected.id}`, { name: form.name, status: form.status })
      }
      setModal(null)
      load()
    } catch (err) {
      setError(err.message)
    }
  }

  const cols = [
    { key: 'id',      label: 'ID',      render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'name',    label: 'Name' },
    { key: 'tax_id',  label: 'Tax ID' },
    { key: 'status',  label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
    { key: 'created_at', label: 'Created', render: (r) => new Date(r.created_at).toLocaleDateString('th-TH') },
  ]

  return (
    <div>
      <PageHeader
        title="Tenant Management"
        action={<Btn onClick={openCreate}>+ เพิ่ม Tenant</Btn>}
      />
      <Table columns={cols} data={data} onRowClick={openEdit} />

      {modal && (
        <Modal title={modal === 'create' ? 'เพิ่ม Tenant' : 'แก้ไข Tenant'} onClose={() => setModal(null)}>
          <form onSubmit={submit}>
            <Input label="ชื่อบริษัท" name="name" value={form.name} onChange={onChange} required />
            {modal === 'create' && (
              <Input label="เลขผู้เสียภาษี (13 หลัก)" name="tax_id" value={form.tax_id} onChange={onChange} required />
            )}
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
