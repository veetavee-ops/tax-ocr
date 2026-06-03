import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, StatusBadge, useForm } from '../components/ui'

export default function Reviewers() {
  const [data, setData]         = useState([])
  const [modal, setModal]       = useState(null)
  const [selected, setSelected] = useState(null)
  const [form, onChange, reset, setForm] = useForm({ name: '', line_user_id: '', reviewer_type: 'text_verifier' })
  const [error, setError]       = useState('')

  const load = () => api.get('/reviewers').then((r) => setData(r.data.data ?? []))
  useEffect(() => { load() }, [])

  const openCreate = () => { reset(); setError(''); setModal('create') }
  const openEdit   = (row) => {
    setForm({ name: row.name, reviewer_type: row.reviewer_type, status: row.status })
    setSelected(row); setError(''); setModal('edit')
  }

  const submit = async (e) => {
    e.preventDefault(); setError('')
    try {
      if (modal === 'create') await api.post('/reviewers', form)
      else await api.put(`/reviewers/${selected.id}`, { name: form.name, reviewer_type: form.reviewer_type, status: form.status })
      setModal(null); load()
    } catch (err) { setError(err.message) }
  }

  const cols = [
    { key: 'id',            label: 'ID',    render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'name',          label: 'ชื่อ' },
    { key: 'reviewer_type', label: 'ประเภท' },
    { key: 'status',        label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
    { key: 'total_earned',  label: 'รายได้รวม', render: (r) => `฿${r.total_earned?.toLocaleString()}` },
    { key: 'pending_payout',label: 'รอจ่าย',   render: (r) => `฿${r.pending_payout?.toLocaleString()}` },
  ]

  return (
    <div>
      <PageHeader title="Reviewers" action={<Btn onClick={openCreate}>+ เพิ่ม Reviewer</Btn>} />
      <Table columns={cols} data={data} onRowClick={openEdit} />

      {modal && (
        <Modal title={modal === 'create' ? 'เพิ่ม Reviewer' : 'แก้ไข Reviewer'} onClose={() => setModal(null)}>
          <form onSubmit={submit}>
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-1">ชื่อ <span className="text-red-500">*</span></label>
              <input name="name" value={form.name} onChange={onChange} required
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm" />
            </div>
            {modal === 'create' && (
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">LINE User ID <span className="text-red-500">*</span></label>
                <input name="line_user_id" value={form.line_user_id} onChange={onChange} required
                  className="w-full border border-gray-300 rounded px-3 py-2 text-sm" />
              </div>
            )}
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-1">ประเภท</label>
              <select name="reviewer_type" value={form.reviewer_type} onChange={onChange}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
                <option value="text_verifier">Text Verifier</option>
                <option value="classification_verifier">Classification Verifier</option>
              </select>
            </div>
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
