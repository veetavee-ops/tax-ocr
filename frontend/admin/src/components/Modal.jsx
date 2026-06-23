import { useEffect, useRef } from 'react'

export default function Modal({ title, onClose, children, devLabel, hideClose }) {
  const contentRef = useRef(null)

  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape') { onClose(); return }
      if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
        if (!contentRef.current) return
        const btns = Array.from(contentRef.current.querySelectorAll('button:not([disabled]):not(.modal-close)'))
        if (btns.length < 2) return
        e.preventDefault()
        const idx = btns.indexOf(document.activeElement)
        if (idx === -1) {
          // focus อยู่ใน modal แต่ไม่ใช่ปุ่ม (เช่น textarea) → jump ไป cancel
          if (contentRef.current.contains(document.activeElement)) btns[0]?.focus()
          return
        }
        const next = e.key === 'ArrowRight' ? (idx + 1) % btns.length : (idx - 1 + btns.length) % btns.length
        btns[next]?.focus()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={onClose}>
      <div ref={contentRef} className="modal-content bg-white rounded-lg shadow-xl w-full max-w-md mx-4"
        onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-base font-semibold text-gray-800">{title}</h2>
          <div className="flex items-center gap-2">
            {devLabel && <span className="text-[10px] font-mono bg-black/70 text-white px-1.5 py-0.5 rounded select-none">{devLabel}</span>}
            {!hideClose && <button onClick={onClose} className="modal-close text-gray-400 hover:text-gray-600 text-xl leading-none">&times;</button>}
          </div>
        </div>
        <div className="px-6 py-4">{children}</div>
      </div>
    </div>
  )
}
