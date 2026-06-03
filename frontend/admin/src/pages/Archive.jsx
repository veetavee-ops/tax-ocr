import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, StatusBadge, useForm } from '../components/ui'

export default function Archive() {
  const [archives, setArchives] = useState([])
  const [policies, setPolicies] = useState([])
  const [tab, setTab]           = useState('archives')
  const [tenants, setTenants]   = useState([])
  const [modal, setModal]       = useState(false)
  const [form, onChange, reset] = useForm({ tenant_id: '', active_days: '90', archive_days: '365' })
  const [msg, setMsg]           = useState('')

  const load = () => {
    api.get('/archive').then((r) => setArchives(r.data.data ?? []))
    api.get('/archive/policies').then((r) => setPolicies(r.data.data ?? []))
    api.get('/tenants').then((r) => setTenants(r.data.data ?? []))
  }
  useEffect(() => { load() }, [])

  const restore = async (id) => {
    await api.post(`/archive/${id}/restore`, {})
    load()
  }

  const submitPolicy = async (e) => {
    e.preventDefault(); setMsg('')
    try {
      await api.post('/archive/policies', { ...form, active_days: parseInt(form.active_days), archive_days: parseInt(form.archive_days) })
      setModal(false); reset(); load()
    } catch (err) { setMsg(err.message) }
  }

  const tenantMap = Object.fromEntries(tenants.map((t) => [t.id, t.name]))

  const archiveCols = [
    { key: 'id', label: 'ID', render: (r) => <span className="font-mono text-xs text-gray-400">{r.id?.slice(0,8)}…</span> },
    { key: 'entity_type', label: 'ประเภท' },
    { key: 'entity_id',   label: 'Entity ID', render: (r) => <span className="font-mono text-xs">{r.entity_id?.slice(0,8)}…</span> },
    { key: 'status',      label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
    { key: 'archived_at', label: 'วันที่ archive', render: (r) => new Date(r.archived_at).toLocaleDateString('th-TH') },
    { key: 'actions', label: '', render: (r) => r.status === 'archived' ? (
      <button onClick={() => restore(r.id)} className="text-xs text-blue-600 hover:underline">Restore</button>
    ) : null },
  ]

  const policyCols = [
    { key: 'tenant_id',   label: 'Tenant', render: (r) => tenantMap[r.tenant_id] ?? r.tenant_id },
    { key: 'active_days', label: 'Active (วัน)' },
    { key: 'archive_days',label: 'Archive (วัน)' },
    { key: 'updated_at',  label: 'อัปเดต', render: (r) => new Date(r.updated_at).toLocaleDateString('th-TH') },
  ]

  return (
    <div>
      <PageHeader title="Archive" action={tab === 'policies' && <Btn onClick={() => { reset(); setMsg(''); setModal(true) }}>+ เพิ่ม Policy</Btn>} />
      <div className="flex gap-2 mb-4">
        {['archives','policies'].map((t) => (
          <button key={t} onClick={() => setTab(t)}
            className={`px-3 py-1 rounded text-sm border ${tab === t ? 'bg-blue-600 text-white border-blue-600' : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'}`}>
            {t === 'archives' ? 'Archive Logs' : 'Policies'}
          </button>
        ))}
      </div>

      {tab === 'archives' ? <Table columns={archiveCols} data={archives} /> : <Table columns={policyCols} data={policies} />}

      {modal && (
        <Modal title="เพิ่ม Archive Policy" onClose={() => setModal(false)}>
          <form onSubmit={submitPolicy}>
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-1">Tenant</label>
              <select name="tenant_id" value={form.tenant_id} onChange={onChange}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
                <option value="">— เลือก —</option>
                {tenants.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
            </div>
            <div className="mb-4"><label className="block text-sm font-medium text-gray-700 mb-1">Active Days</label>
              <input name="active_days" value={form.active_days} onChange={onChange} type="number" className="w-full border border-gray-300 rounded px-3 py-2 text-sm" /></div>
            <div className="mb-4"><label className="block text-sm font-medium text-gray-700 mb-1">Archive Days</label>
              <input name="archive_days" value={form.archive_days} onChange={onChange} type="number" className="w-full border border-gray-300 rounded px-3 py-2 text-sm" /></div>
            {msg && <p className="text-red-500 text-sm mb-3">{msg}</p>}
            <div className="flex justify-end gap-2">
              <Btn variant="secondary" onClick={() => setModal(false)}>ยกเลิก</Btn>
              <Btn type="submit">บันทึก</Btn>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
