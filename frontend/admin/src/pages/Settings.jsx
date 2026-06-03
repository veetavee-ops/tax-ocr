import { useEffect, useState } from 'react'
import api from '../api/client'
import { PageHeader, Btn } from '../components/ui'

export default function Settings() {
  const [rewards, setRewards]   = useState([])
  const [editing, setEditing]   = useState({})
  const [msg, setMsg]           = useState('')

  const load = () => api.get('/reward/config').then((r) => {
    const data = r.data.data ?? []
    setRewards(data)
    setEditing(Object.fromEntries(data.map((d) => [d.id, String(d.amount)])))
  })
  useEffect(() => { load() }, [])

  const save = async (id) => {
    setMsg('')
    try {
      await api.put(`/reward/config/${id}`, { amount: parseFloat(editing[id]) })
      setMsg('บันทึกสำเร็จ')
      load()
    } catch (err) { setMsg(err.message) }
  }

  return (
    <div>
      <PageHeader title="Settings" />

      <div className="bg-white rounded-lg shadow p-6 max-w-lg">
        <h3 className="font-semibold text-gray-700 mb-4">Reward Config</h3>

        {rewards.length === 0
          ? <p className="text-sm text-gray-400">ยังไม่มีข้อมูล reward config</p>
          : rewards.map((r) => (
            <div key={r.id} className="flex items-center gap-3 mb-3">
              <span className="text-sm text-gray-600 w-56">{r.task_type}</span>
              <input
                type="number"
                value={editing[r.id] ?? r.amount}
                onChange={(e) => setEditing((prev) => ({ ...prev, [r.id]: e.target.value }))}
                className="border border-gray-300 rounded px-3 py-1.5 text-sm w-24"
              />
              <span className="text-sm text-gray-400">{r.currency}</span>
              <Btn onClick={() => save(r.id)}>บันทึก</Btn>
            </div>
          ))
        }

        {msg && (
          <p className={`text-sm mt-3 ${msg === 'บันทึกสำเร็จ' ? 'text-green-600' : 'text-red-500'}`}>{msg}</p>
        )}
      </div>
    </div>
  )
}
