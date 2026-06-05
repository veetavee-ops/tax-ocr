import React, { useEffect, useState, useCallback } from 'react'
import client from '../api/client.js'

const STATUS_LABEL = {
  pending:    { label: 'รอประมวลผล',  color: 'bg-yellow-100 text-yellow-800' },
  processing: { label: 'กำลังประมวลผล', color: 'bg-blue-100 text-blue-800' },
  done:       { label: 'เสร็จแล้ว',    color: 'bg-green-100 text-green-800' },
  failed:     { label: 'ล้มเหลว',      color: 'bg-red-100 text-red-800' },
}

const SOURCE_ICON = {
  camera: '📷',
  upload: '🖼',
  zip:    '🗜',
}

function DocCard({ doc }) {
  const s = STATUS_LABEL[doc.status] || { label: doc.status, color: 'bg-gray-100 text-gray-700' }
  const date = new Date(doc.created_at).toLocaleString('th-TH', {
    day: '2-digit', month: 'short', year: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4">
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-xl">{SOURCE_ICON[doc.source_type] || '📄'}</span>
          <div className="min-w-0">
            <p className="text-sm font-medium text-gray-800 capitalize truncate">
              {doc.source_type}
            </p>
            <p className="text-xs text-gray-400">{date}</p>
          </div>
        </div>
        <span className={`text-xs font-medium px-2.5 py-1 rounded-full whitespace-nowrap ${s.color}`}>
          {s.label}
        </span>
      </div>
      {doc.total_files > 0 && (
        <div className="mt-3">
          <div className="flex justify-between text-xs text-gray-500 mb-1">
            <span>ประมวลผลแล้ว</span>
            <span>{doc.processed_files}/{doc.total_files} ไฟล์</span>
          </div>
          <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
            <div
              className="h-full bg-line rounded-full transition-all"
              style={{ width: `${doc.total_files > 0 ? (doc.processed_files / doc.total_files) * 100 : 0}%` }}
            />
          </div>
        </div>
      )}
    </div>
  )
}

export default function Status() {
  const [docs, setDocs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    client.get('/my-documents')
      .then(res => setDocs(res.data.data || []))
      .catch(err => setError(err.response?.data?.error || 'โหลดข้อมูลล้มเหลว'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
    // Auto-refresh every 10 seconds if any doc is pending/processing
    const interval = setInterval(() => {
      setDocs(prev => {
        const hasActive = prev.some(d => d.status === 'pending' || d.status === 'processing')
        if (hasActive) load()
        return prev
      })
    }, 10000)
    return () => clearInterval(interval)
  }, [load])

  return (
    <div className="px-4 py-6 max-w-md mx-auto">
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-xl font-bold text-gray-800">สถานะเอกสาร</h1>
        <button
          onClick={load}
          className="text-sm text-line font-medium flex items-center gap-1"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          รีเฟรช
        </button>
      </div>

      {loading ? (
        <div className="flex justify-center py-16">
          <div className="w-8 h-8 border-4 border-line border-t-transparent rounded-full animate-spin" />
        </div>
      ) : error ? (
        <div className="text-center py-12">
          <p className="text-red-600 text-sm mb-3">{error}</p>
          <button onClick={load} className="text-line text-sm underline">ลองใหม่</button>
        </div>
      ) : docs.length === 0 ? (
        <div className="text-center py-16">
          <div className="text-5xl mb-3">📂</div>
          <p className="text-gray-500 text-sm">ยังไม่มีเอกสารที่ส่ง</p>
        </div>
      ) : (
        <div className="space-y-3">
          {docs.map(doc => <DocCard key={doc.id} doc={doc} />)}
        </div>
      )}
    </div>
  )
}
