import React, { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useLiff } from '../contexts/LiffContext.jsx'

export default function Login() {
  const { ready, loggedIn, triggerLineLogin, error, tenant_id } = useLiff()
  const navigate = useNavigate()

  useEffect(() => {
    if (loggedIn) navigate('/upload', { replace: true })
  }, [loggedIn, navigate])

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-white px-6">
      <div className="w-20 h-20 bg-line rounded-2xl flex items-center justify-center mb-6 shadow-lg">
        <span className="text-white text-4xl">🧾</span>
      </div>

      <h1 className="text-2xl font-bold text-gray-800 mb-1">Tax OCR</h1>
      <p className="text-gray-500 text-sm mb-8 text-center">ส่งใบกำกับภาษีได้ง่าย ๆ<br />ผ่าน LINE</p>

      {!ready ? (
        <div className="w-8 h-8 border-4 border-line border-t-transparent rounded-full animate-spin" />
      ) : (
        <>
          {!tenant_id && (
            <p className="text-amber-600 text-sm mb-4 text-center bg-amber-50 rounded-lg px-4 py-2">
              กรุณาเปิดลิงก์นี้จาก LINE ที่บริษัทแจกให้
            </p>
          )}

          {error && (
            <p className="text-red-600 text-sm mb-4 text-center bg-red-50 rounded-lg px-4 py-2">
              {error}
            </p>
          )}

          <button
            onClick={triggerLineLogin}
            disabled={!tenant_id}
            className="w-full bg-line hover:bg-line-dark disabled:opacity-50 text-white font-semibold py-3.5 rounded-xl flex items-center justify-center gap-2 transition-colors shadow"
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2C6.48 2 2 6.05 2 11.05c0 4.5 3.79 8.27 8.92 8.93.35.07.82.22.94.5.11.26.07.66.03.92l-.15.9c-.05.26-.22 1.03.9.56 1.12-.47 6.07-3.57 8.28-6.11C22.33 14.8 22 12.99 22 11.05 22 6.05 17.52 2 12 2z"/>
            </svg>
            เข้าสู่ระบบด้วย LINE
          </button>

          <p className="text-xs text-gray-400 mt-6 text-center">
            ระบบจัดการใบกำกับภาษีอิเล็กทรอนิกส์
          </p>
        </>
      )}
    </div>
  )
}
