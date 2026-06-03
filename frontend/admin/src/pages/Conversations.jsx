import { useEffect, useState } from 'react'
import api from '../api/client'
import { PageHeader, StatusBadge } from '../components/ui'

export default function Conversations() {
  const [data, setData]     = useState([])
  const [selected, setSelected] = useState(null)
  const [messages, setMessages] = useState([])
  const [reply, setReply]   = useState('')

  useEffect(() => {
    api.get('/conversations').then((r) => setData(r.data.data ?? []))
  }, [])

  const open = (conv) => {
    setSelected(conv)
    api.get(`/conversations/${conv.id}/messages`).then((r) => setMessages(r.data.data ?? []))
  }

  const send = async (e) => {
    e.preventDefault()
    if (!reply.trim()) return
    await api.post(`/conversations/${selected.id}/messages`, {
      sender_type: 'admin',
      message_type: 'text',
      content: reply,
    })
    setReply('')
    api.get(`/conversations/${selected.id}/messages`).then((r) => setMessages(r.data.data ?? []))
  }

  return (
    <div className="flex gap-4 h-full">
      {/* Conversation list */}
      <div className="w-64 bg-white rounded-lg shadow overflow-y-auto flex-shrink-0">
        <div className="px-4 py-3 border-b font-semibold text-sm text-gray-700">Conversations</div>
        {data.length === 0
          ? <p className="p-4 text-sm text-gray-400">ไม่มีการสนทนา</p>
          : data.map((c) => (
            <button key={c.id} onClick={() => open(c)}
              className={`w-full text-left px-4 py-3 text-sm border-b hover:bg-blue-50 transition-colors ${selected?.id === c.id ? 'bg-blue-50' : ''}`}>
              <p className="font-medium text-gray-700 truncate">{c.line_user_id || c.id.slice(0,8)}</p>
              <div className="flex items-center justify-between mt-0.5">
                <span className="text-xs text-gray-400">{c.channel}</span>
                <StatusBadge value={c.status} />
              </div>
            </button>
          ))
        }
      </div>

      {/* Chat panel */}
      <div className="flex-1 bg-white rounded-lg shadow flex flex-col">
        {!selected ? (
          <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">เลือกการสนทนา</div>
        ) : (
          <>
            <div className="px-4 py-3 border-b text-sm font-semibold text-gray-700">
              {selected.line_user_id || selected.id} — <StatusBadge value={selected.status} />
            </div>
            <div className="flex-1 overflow-y-auto p-4 space-y-3">
              {messages.map((m) => (
                <div key={m.id} className={`flex ${m.sender_type === 'admin' ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-xs px-3 py-2 rounded-lg text-sm ${
                    m.sender_type === 'admin' ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-800'
                  }`}>
                    {m.content}
                    <p className="text-xs mt-1 opacity-60">{m.sender_type}</p>
                  </div>
                </div>
              ))}
              {messages.length === 0 && <p className="text-center text-gray-400 text-sm">ยังไม่มีข้อความ</p>}
            </div>
            <form onSubmit={send} className="px-4 py-3 border-t flex gap-2">
              <input value={reply} onChange={(e) => setReply(e.target.value)} placeholder="พิมพ์ข้อความ…"
                className="flex-1 border border-gray-300 rounded px-3 py-2 text-sm" />
              <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">ส่ง</button>
            </form>
          </>
        )}
      </div>
    </div>
  )
}
