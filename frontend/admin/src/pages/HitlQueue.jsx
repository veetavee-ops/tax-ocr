import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import { PageHeader, Btn, StatusBadge } from '../components/ui'

export default function HitlQueue() {
  const [data, setData]   = useState([])
  const [filter, setFilter] = useState('pending')
  const [error, setError] = useState('')

  const load = () =>
    api.get(`/hitl/queue${filter ? `?status=${filter}` : ''}`).then((r) => setData(r.data.data ?? []))

  useEffect(() => { load() }, [filter])

  const resolve = async (id) => {
    setError('')
    try {
      await api.post(`/hitl/${id}/resolve`, { resolved_by: '' })
      load()
    } catch (e) { setError(e.message) }
  }

  const reject = async (id) => {
    setError('')
    try {
      await api.post(`/hitl/${id}/reject`, {})
      load()
    } catch (e) { setError(e.message) }
  }

  const cols = [
    { key: 'id',     label: 'ID', render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'invoice_item_id', label: 'Item ID', render: (r) => <span className="font-mono text-xs">{r.invoice_item_id.slice(0,8)}…</span> },
    { key: 'reason', label: 'เหตุผล' },
    { key: 'status', label: 'Status', render: (r) => <StatusBadge value={r.status} /> },
    { key: 'created_at', label: 'วันที่', render: (r) => new Date(r.created_at).toLocaleDateString('th-TH') },
    { key: 'actions', label: '', render: (r) => r.status === 'pending' ? (
      <div className="flex gap-2">
        <button onClick={(e) => { e.stopPropagation(); resolve(r.id) }}
          className="text-xs text-green-600 hover:underline">Resolve</button>
        <button onClick={(e) => { e.stopPropagation(); reject(r.id) }}
          className="text-xs text-red-500 hover:underline">Reject</button>
      </div>
    ) : null },
  ]

  return (
    <div>
      <PageHeader title="HITL Queue" />
      <div className="flex gap-2 mb-4">
        {['', 'pending', 'resolved'].map((s) => (
          <button key={s} onClick={() => setFilter(s)}
            className={`px-3 py-1 rounded text-sm border ${filter === s ? 'bg-blue-600 text-white border-blue-600' : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'}`}>
            {s || 'ทั้งหมด'}
          </button>
        ))}
      </div>
      {error && <p className="text-red-500 text-sm mb-3">{error}</p>}
      <Table columns={cols} data={data} />
    </div>
  )
}
