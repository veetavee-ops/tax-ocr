import React, { useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useLiff } from '../contexts/LiffContext.jsx'
import client from '../api/client.js'

const MAX_SIZE = 20 * 1024 * 1024 // 20 MB

export default function Upload() {
  const { user } = useLiff()
  const navigate = useNavigate()
  const cameraRef = useRef(null)
  const fileRef = useRef(null)

  const zipRef = useRef(null)
  const [tab, setTab] = useState('camera')
  const [preview, setPreview] = useState(null)
  const [file, setFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)

  const branchId = localStorage.getItem('liff_branch')
  const branchName = localStorage.getItem('liff_branch_name')

  const handleFile = (e) => {
    const f = e.target.files?.[0]
    if (!f) return
    if (f.size > MAX_SIZE) {
      setError('ไฟล์ขนาดใหญ่เกินไป (สูงสุด 20 MB)')
      return
    }
    setError(null)
    setFile(f)
    setResult(null)
    if (f.type.startsWith('image/')) {
      const url = URL.createObjectURL(f)
      setPreview(url)
    } else {
      setPreview(null)
    }
  }

  const handleUpload = async () => {
    if (!file || !user || !branchId) return
    setUploading(true)
    setError(null)
    try {
      const form = new FormData()
      form.append('file', file)
      form.append('tenant_id', user.tenant_id)
      form.append('branch_id', branchId)
      form.append('user_id', user.id)
      const endpoint = tab === 'zip' ? '/documents/zip' : '/documents/upload'
      const res = await client.post(endpoint, form)
      setResult(res.data)
      setFile(null)
      setPreview(null)
    } catch (err) {
      setError(err.response?.data?.error || 'อัพโหลดล้มเหลว กรุณาลองใหม่')
    } finally {
      setUploading(false)
    }
  }

  const reset = () => {
    setFile(null)
    setPreview(null)
    setResult(null)
    setError(null)
  }

  if (!branchId) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[70vh] px-6">
        <p className="text-gray-600 text-center mb-4">กรุณาเลือกสาขาก่อนส่งเอกสาร</p>
        <button
          onClick={() => navigate('/branch')}
          className="bg-line text-white px-6 py-2.5 rounded-xl font-semibold"
        >
          เลือกสาขา
        </button>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 max-w-md mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-bold text-gray-800">ส่งเอกสาร</h1>
          <p className="text-xs text-gray-500 mt-0.5">สาขา: {branchName}</p>
        </div>
        <button
          onClick={() => navigate('/branch')}
          className="text-xs text-line underline"
        >
          เปลี่ยนสาขา
        </button>
      </div>

      {/* Tabs */}
      <div className="flex rounded-xl bg-gray-100 p-1 mb-5">
        {[
          { key: 'camera', label: '📷 กล้อง' },
          { key: 'file',   label: '🖼 ไฟล์' },
          { key: 'zip',    label: '🗜 ZIP' },
        ].map(t => (
          <button
            key={t.key}
            onClick={() => { setTab(t.key); reset() }}
            className={`flex-1 py-2 text-sm font-medium rounded-lg transition-all ${
              tab === t.key ? 'bg-white shadow text-gray-800' : 'text-gray-500'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Success state */}
      {result ? (
        <div className="text-center py-8">
          <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 className="text-lg font-bold text-gray-800 mb-1">อัพโหลดสำเร็จ!</h2>
          <p className="text-sm text-gray-500 mb-6">กำลังประมวลผล OCR ในพื้นหลัง</p>
          <div className="space-y-3">
            <button
              onClick={() => navigate('/status')}
              className="w-full bg-line text-white font-semibold py-3 rounded-xl"
            >
              ดูสถานะเอกสาร
            </button>
            <button
              onClick={reset}
              className="w-full border border-gray-300 text-gray-700 font-semibold py-3 rounded-xl"
            >
              ส่งเอกสารอีกชิ้น
            </button>
          </div>
        </div>
      ) : (
        <>
          {/* File picker area */}
          <div
            onClick={() => (tab === 'camera' ? cameraRef : tab === 'zip' ? zipRef : fileRef).current?.click()}
            className="border-2 border-dashed border-gray-300 rounded-2xl flex flex-col items-center justify-center min-h-[200px] cursor-pointer hover:border-line transition-colors bg-white mb-4"
          >
            {preview ? (
              <img src={preview} alt="preview" className="max-h-64 max-w-full rounded-xl object-contain" />
            ) : file ? (
              <div className="text-center px-4">
                <div className="text-4xl mb-2">📄</div>
                <p className="text-sm font-medium text-gray-700">{file.name}</p>
                <p className="text-xs text-gray-400 mt-1">{(file.size / 1024).toFixed(0)} KB</p>
              </div>
            ) : (
              <div className="text-center px-6">
                <div className="text-5xl mb-3">{tab === 'camera' ? '📷' : tab === 'zip' ? '🗜' : '🖼'}</div>
                <p className="text-sm font-medium text-gray-700">
                  {tab === 'camera' ? 'แตะเพื่อถ่ายรูปใบกำกับภาษี'
                    : tab === 'zip' ? 'แตะเพื่อเลือกไฟล์ ZIP'
                    : 'แตะเพื่อเลือกไฟล์ภาพหรือ PDF'}
                </p>
                <p className="text-xs text-gray-400 mt-1">
                  {tab === 'zip' ? 'ZIP ที่มีไฟล์ JPG/PNG/PDF — สูงสุด 100 MB' : 'JPG, PNG, PDF — สูงสุด 20 MB'}
                </p>
              </div>
            )}
          </div>

          {/* Hidden inputs */}
          <input ref={cameraRef} type="file" accept="image/*" capture="environment" className="hidden" onChange={handleFile} />
          <input ref={fileRef}   type="file" accept="image/*,.pdf"                   className="hidden" onChange={handleFile} />
          <input ref={zipRef}    type="file" accept=".zip"                            className="hidden" onChange={handleFile} />

          {error && (
            <div className="bg-red-50 text-red-700 text-sm rounded-lg px-4 py-2.5 mb-4">
              {error}
            </div>
          )}

          {file && !uploading && (
            <button
              onClick={() => (tab === 'camera' ? cameraRef : tab === 'zip' ? zipRef : fileRef).current?.click()}
              className="w-full border border-gray-300 text-gray-600 py-2.5 rounded-xl text-sm mb-3"
            >
              เปลี่ยนไฟล์
            </button>
          )}

          <button
            onClick={handleUpload}
            disabled={!file || uploading}
            className="w-full bg-line hover:bg-line-dark disabled:opacity-40 text-white font-semibold py-3.5 rounded-xl flex items-center justify-center gap-2 transition-colors"
          >
            {uploading ? (
              <>
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                กำลังอัพโหลด...
              </>
            ) : (
              '📤 ส่งเอกสาร'
            )}
          </button>
        </>
      )}
    </div>
  )
}
