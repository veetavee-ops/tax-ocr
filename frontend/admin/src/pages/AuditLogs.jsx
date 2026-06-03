import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import { PageHeader } from '../components/ui'

export default function AuditLogs() {
  const [data, setData]     = useState([])
  const [tenants, setTenants] = useState([])
  const [tenantID, setTenantID] = useState('')

  const load = (tid) =>
    api.get(`/audit-logs${tid ? `?tenant_id=${tid}` : ''}`).then((r) => setData(r.data.data ?? []))

  useEffect(() => {
    api.get('/tenants').then((r) => setTenants(r.data.data ?? []))
    load('')
  }, [])

  const cols = [
    { key: 'id',          label: 'ID',     render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'action',      label: 'Action' },
    { key: 'entity_type', label: 'Entity Type' },
    { key: 'entity_id',   label: 'Entity ID', render: (r) => r.entity_id ? <span className="font-mono text-xs">{r.entity_id.slice(0,8)}…</span> : '—' },
    { key: 'user_id',     label: 'User',   render: (r) => r.user_id ? <span className="font-mono text-xs">{r.user_id.slice(0,8)}…</span> : '—' },
    { key: 'ip_address',  label: 'IP' },
    { key: 'created_at',  label: 'วันที่', render: (r) => new Date(r.created_at).toLocaleString('th-TH') },
  ]

  return (
    <div>
      <PageHeader title="Audit Logs" />
      <div className="flex gap-2 mb-4 items-center">
        <span className="text-sm text-gray-600">Tenant:</span>
        <select value={tenantID} onChange={(e) => { setTenantID(e.target.value); load(e.target.value) }}
          className="border border-gray-300 rounded px-3 py-2 text-sm">
          <option value="">ทั้งหมด</option>
          {tenants.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
        </select>
        <span className="text-sm text-gray-400 ml-2">{data.length} รายการ</span>
      </div>
      <Table columns={cols} data={data} />
    </div>
  )
}
