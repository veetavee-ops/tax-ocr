import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useLiff } from '../contexts/LiffContext.jsx'
import client from '../api/client.js'

export default function BranchSelect() {
  const { user } = useLiff()
  const navigate = useNavigate()
  const [branches, setBranches] = useState([])
  const [selected, setSelected] = useState(localStorage.getItem('liff_branch') || '')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user?.tenant_id) return
    client.get(`/tenants/${user.tenant_id}/branches`)
      .then(res => {
        const list = res.data.data || []
        setBranches(list.filter(b => b.status === 'active'))
        // Auto-select if only one branch
        if (list.length === 1 && !selected) {
          const b = list[0]
          setSelected(b.id)
          localStorage.setItem('liff_branch', b.id)
          localStorage.setItem('liff_branch_name', b.name)
        }
      })
      .finally(() => setLoading(false))
  }, [user])

  const handleSelect = (branch) => {
    setSelected(branch.id)
    localStorage.setItem('liff_branch', branch.id)
    localStorage.setItem('liff_branch_name', branch.name)
  }

  const handleConfirm = () => {
    if (selected) navigate('/upload')
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="w-8 h-8 border-4 border-line border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  return (
    <div className="px-4 py-6 max-w-md mx-auto">
      <h1 className="text-xl font-bold text-gray-800 mb-1">เลือกสาขา</h1>
      <p className="text-sm text-gray-500 mb-6">เลือกสาขาที่คุณต้องการส่งเอกสาร</p>

      {branches.length === 0 ? (
        <div className="text-center text-gray-400 py-12">ไม่พบสาขาที่เปิดใช้งาน</div>
      ) : (
        <div className="space-y-3 mb-8">
          {branches.map(b => (
            <button
              key={b.id}
              onClick={() => handleSelect(b)}
              className={`w-full text-left p-4 rounded-xl border-2 transition-all ${
                selected === b.id
                  ? 'border-line bg-green-50'
                  : 'border-gray-200 bg-white'
              }`}
            >
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-semibold text-gray-800">{b.name}</p>
                  {b.code && <p className="text-xs text-gray-500 mt-0.5">รหัส: {b.code}</p>}
                </div>
                {selected === b.id && (
                  <div className="w-6 h-6 rounded-full bg-line flex items-center justify-center">
                    <svg className="w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                )}
              </div>
            </button>
          ))}
        </div>
      )}

      <button
        onClick={handleConfirm}
        disabled={!selected}
        className="w-full bg-line hover:bg-line-dark disabled:opacity-40 text-white font-semibold py-3.5 rounded-xl transition-colors"
      >
        ยืนยัน
      </button>
    </div>
  )
}
