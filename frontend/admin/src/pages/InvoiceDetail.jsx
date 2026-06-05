import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import api from '../api/client'
import Table from '../components/Table'
import { Btn, StatusBadge } from '../components/ui'
import VerificationWizard from './VerificationWizard'

/* ─────────────────────────────────────────────
   Shared interactive image core
   - scroll wheel  → zoom
   - drag          → pan
   - rotate buttons → rotate
   - onDoubleClick → optional callback (open popup)
───────────────────────────────────────────── */
function ImageInteractive({ url, isPdf, height = '100%', onDoubleClick, onClose }) {
  const [scale, setScale]         = useState(1)
  const [rotate, setRotate]       = useState(0)
  const [translate, setTranslate] = useState({ x: 0, y: 0 })
  const [panning, setPanning]     = useState(false)
  const panStart    = useRef({ mx: 0, my: 0, tx: 0, ty: 0 })
  const containerRef = useRef(null)
  const imgRef       = useRef(null)
  const fitScaleRef  = useRef(1)

  // Scale to contain (fit whole image inside container — like object-fit: contain)
  const applyFitScale = () => {
    const container = containerRef.current
    if (!container) return
    const cw = container.clientWidth
    const ch = container.clientHeight
    const s = isPdf
      ? Math.min(cw / 680, ch / 880)
      : imgRef.current?.naturalWidth
        ? Math.min(cw / imgRef.current.naturalWidth, ch / imgRef.current.naturalHeight)
        : 1
    fitScaleRef.current = s
    setScale(s)
    setTranslate({ x: 0, y: 0 })
  }

  // PDF: fit on mount; image: fit on load (onImgLoad below)
  useEffect(() => { if (isPdf) applyFitScale() }, [isPdf])

  // Wheel zoom (non-passive)
  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const onWheel = (e) => {
      e.preventDefault()
      setScale((s) => Math.min(Math.max(s * (e.deltaY < 0 ? 1.12 : 0.9), 0.1), 12))
    }
    el.addEventListener('wheel', onWheel, { passive: false })
    return () => el.removeEventListener('wheel', onWheel)
  }, [])

  // Pan via window-level listeners so fast mouse movement stays tracked
  useEffect(() => {
    if (!panning) return
    const onMove = (e) =>
      setTranslate({
        x: panStart.current.tx + (e.clientX - panStart.current.mx),
        y: panStart.current.ty + (e.clientY - panStart.current.my),
      })
    const onUp = () => setPanning(false)
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
  }, [panning])

  // Esc: close popup (if onClose provided) or reset to fit-width (inline mode)
  useEffect(() => {
    const onKey = (e) => {
      if (e.key !== 'Escape') return
      if (onClose) onClose()
      else { setScale(fitScaleRef.current); setRotate(0); setTranslate({ x: 0, y: 0 }) }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  const onMouseDown = (e) => {
    if (e.button !== 0) return
    e.preventDefault()
    panStart.current = { mx: e.clientX, my: e.clientY, tx: translate.x, ty: translate.y }
    setPanning(true)
  }

  const onImgLoad = () => applyFitScale()

  const reset = () => {
    setScale(fitScaleRef.current)
    setRotate(0)
    setTranslate({ x: 0, y: 0 })
  }

  const btnCls = 'px-2.5 py-1 rounded text-xs font-medium bg-gray-800 hover:bg-gray-700 text-white transition-colors select-none'

  const transform = `translate(${translate.x}px, ${translate.y}px) scale(${scale}) rotate(${rotate}deg)`

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height, background: '#0f172a', borderRadius: 'inherit' }}>

      {/* Toolbar */}
      <div style={{ display: 'flex', gap: 5, padding: '6px 10px', alignItems: 'center', flexWrap: 'wrap' }}>
        <button className={btnCls} onClick={() => setScale((s) => Math.min(s * 1.25, 12))}>🔍+</button>
        <button className={btnCls} onClick={() => setScale((s) => Math.max(s * 0.8, 0.1))}>🔍−</button>
        <button className={btnCls} onClick={() => setRotate((r) => r - 90)}>↺ ซ้าย</button>
        <button className={btnCls} onClick={() => setRotate((r) => r + 90)}>↻ ขวา</button>
        <button className={btnCls} onClick={reset} style={{ background: '#374151' }}>⊙ Reset</button>
        <span style={{ color: '#64748b', fontSize: 11, marginLeft: 2 }}>
          {Math.round(scale * 100)}%{rotate !== 0 ? ` · ${rotate}°` : ''}
        </span>
        {onClose && (
          <button
            className={btnCls}
            onClick={onClose}
            style={{ background: '#7f1d1d', marginLeft: 'auto' }}
          >✕ ปิด</button>
        )}
        {onDoubleClick && (
          <span style={{ color: '#475569', fontSize: 11, marginLeft: 'auto' }}>
            ดับเบิ้ลคลิกเพื่อขยาย
          </span>
        )}
      </div>

      {/* Image area */}
      <div
        ref={containerRef}
        onMouseDown={onMouseDown}
        onDoubleClick={onDoubleClick}
        style={{
          overflow: 'hidden',
          flex: 1,
          cursor: panning ? 'grabbing' : 'grab',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: '#f1f5f9',
          userSelect: 'none',
        }}
      >
        {isPdf ? (
          <iframe
            src={url}
            title="invoice"
            style={{
              width: `${Math.round(680 * scale)}px`,
              height: `${Math.round(880 * scale)}px`,
              transform: `translate(${translate.x}px, ${translate.y}px) rotate(${rotate}deg)`,
              transformOrigin: 'center',
              transition: panning ? 'none' : 'transform 0.1s',
              border: 'none',
              pointerEvents: 'none',
              flexShrink: 0,
            }}
          />
        ) : (
          <img
            ref={imgRef}
            src={url}
            alt="invoice"
            draggable={false}
            onLoad={onImgLoad}
            style={{
              maxWidth: 'none',
              transform,
              transformOrigin: 'center',
              transition: panning ? 'none' : 'transform 0.1s',
              display: 'block',
              pointerEvents: 'none',
            }}
          />
        )}
      </div>
    </div>
  )
}

/* ─────────────────────────────────────────────
   Floating popup window — draggable header + ImageInteractive
───────────────────────────────────────────── */
function ImageViewer({ url, isPdf, onClose }) {
  const [pos, setPos]     = useState(() => ({
    x: Math.max(40, window.innerWidth / 2 - 400),
    y: Math.max(40, window.innerHeight / 2 - 340),
  }))
  const [dragging, setDragging] = useState(false)
  const dragOffset = useRef({ x: 0, y: 0 })

  useEffect(() => {
    if (!dragging) return
    const onMove = (e) =>
      setPos({ x: e.clientX - dragOffset.current.x, y: e.clientY - dragOffset.current.y })
    const onUp = () => setDragging(false)
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
  }, [dragging])

  const onHeaderMouseDown = (e) => {
    dragOffset.current = { x: e.clientX - pos.x, y: e.clientY - pos.y }
    setDragging(true)
  }

  return (
    <div
      style={{
        position: 'fixed',
        top: pos.y,
        left: pos.x,
        zIndex: 9999,
        width: 800,
        maxWidth: '96vw',
        borderRadius: 10,
        boxShadow: '0 16px 60px rgba(0,0,0,0.45)',
        overflow: 'hidden',
        userSelect: dragging ? 'none' : 'auto',
      }}
    >
      {/* Drag handle header */}
      <div
        onMouseDown={onHeaderMouseDown}
        style={{
          cursor: dragging ? 'grabbing' : 'grab',
          background: '#1e293b',
          padding: '9px 14px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ color: '#e2e8f0', fontSize: 13, fontWeight: 600 }}>
          🖼 รูปต้นฉบับ — drag header เพื่อย้าย
        </span>
        <button
          onClick={onClose}
          style={{
            background: 'none', border: 'none', color: '#94a3b8',
            fontSize: 20, cursor: 'pointer', lineHeight: 1, padding: '0 4px',
          }}
        >✕</button>
      </div>

      <ImageInteractive url={url} isPdf={isPdf} height="calc(85vh - 60px)" onClose={onClose} />
    </div>
  )
}

/* ─────────────────────────────────────────────
   Invoice Detail Page
───────────────────────────────────────────── */
export default function InvoiceDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [invoice, setInvoice] = useState(null)
  const [items, setItems]     = useState([])
  const [imageUrl, setImageUrl]   = useState(null)
  const [filePath, setFilePath]   = useState('')
  const [imageError, setImageError] = useState(null)
  const [editing, setEditing]   = useState(false)
  const [form, setForm]         = useState({})
  const [saving, setSaving]     = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [reprocessing, setReprocessing] = useState(false)
  const [showViewer, setShowViewer] = useState(false)

  const loadInvoice = () =>
    api.get(`/invoices/${id}`).then((r) => setInvoice(r.data.data))
  const loadItems = () =>
    api.get(`/invoices/${id}/items`).then((r) => setItems(r.data.data ?? []))

  useEffect(() => {
    loadInvoice()
    loadItems()
    let blobUrl = null
    api.get(`/invoices/${id}/image`, { responseType: 'blob' })
      .then((r) => {
        const fp = r.headers['x-file-path'] ?? ''
        blobUrl = URL.createObjectURL(r.data)
        setImageUrl(blobUrl)
        setFilePath(fp)
        setImageError(null)
      })
      .catch(async (err) => {
        let msg = err.message || 'โหลดรูปไม่ได้'
        if (err.response?.data instanceof Blob) {
          try {
            const text = await err.response.data.text()
            const parsed = JSON.parse(text)
            if (parsed.error) msg = parsed.error
          } catch (_) {}
        }
        setImageError(`${err.response?.status ?? ''} ${msg}`.trim())
      })
    return () => { if (blobUrl) URL.revokeObjectURL(blobUrl) }
  }, [id])

  useEffect(() => {
    if (invoice?.status !== 'pending') return
    const timer = setInterval(() => { loadInvoice(); loadItems() }, 3000)
    return () => clearInterval(timer)
  }, [invoice?.status])

  useEffect(() => {
    if (invoice && !editing) setForm(invoice)
  }, [invoice])

  if (!invoice) return <p className="text-gray-400">กำลังโหลด…</p>

  const fmt = (n) => {
    const parts = Number(n ?? 0).toFixed(2).split('.')
    parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    return parts.join('.')
  }

  const isPdf = filePath.toLowerCase().endsWith('.pdf')

  const handleField = (field, val) => setForm((f) => ({ ...f, [field]: val }))
  const startEdit  = () => { setForm({ ...invoice }); setEditing(true) }
  const cancelEdit = () => { setForm({ ...invoice }); setEditing(false) }

  const handleSave = async () => {
    setSaving(true)
    try {
      await api.put(`/invoices/${id}`, {
        vendor_name:      form.vendor_name || '',
        vendor_tax_id:    form.vendor_tax_id || '',
        invoice_doc_no:   form.invoice_doc_no || '',
        invoice_date:     form.invoice_date || '',
        total_before_vat: parseFloat(form.total_before_vat) || 0,
        vat_amount:       parseFloat(form.vat_amount) || 0,
        total_amount:     parseFloat(form.total_amount) || 0,
      })
      setEditing(false)
      loadInvoice()
    } catch (e) {
      alert('บันทึกไม่สำเร็จ: ' + (e.response?.data?.error || e.message))
    } finally {
      setSaving(false)
    }
  }

  const handleReprocess = async () => {
    setReprocessing(true)
    try {
      await api.post(`/invoices/${id}/reprocess`, {})
      setInvoice((inv) => ({ ...inv, status: 'pending' }))
    } catch (e) {
      alert('ส่ง OCR ใหม่ไม่สำเร็จ: ' + (e.response?.data?.error || e.message))
    } finally {
      setReprocessing(false)
    }
  }

  const handleVerify = async () => {
    if (!window.confirm('ยืนยันใบกำกับภาษีนี้? สถานะจะเปลี่ยนเป็น verified')) return
    setVerifying(true)
    try {
      const r = await api.post(`/invoices/${id}/verify`, {})
      setInvoice(r.data.data)
    } catch (e) {
      alert('ยืนยันไม่สำเร็จ: ' + (e.response?.data?.error || e.message))
    } finally {
      setVerifying(false)
    }
  }

  const Field = ({ label, field, mono = false }) => (
    <div>
      <p className="text-xs text-gray-400 mb-0.5">{label}</p>
      {editing ? (
        <input
          value={form[field] ?? ''}
          onChange={(e) => handleField(field, e.target.value)}
          className="border border-gray-300 rounded px-2 py-1 w-full text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
      ) : (
        <p className={`font-medium text-gray-800 ${mono ? 'font-mono' : ''}`}>
          {invoice[field] || '—'}
        </p>
      )}
    </div>
  )

  const itemCols = [
    { key: 'description',   label: 'รายการ' },
    { key: 'quantity',      label: 'จำนวน',     render: (r) => fmt(r.quantity) },
    { key: 'unit_price',    label: 'ราคา/หน่วย', render: (r) => fmt(r.unit_price) },
    { key: 'total_price',   label: 'รวม',        render: (r) => fmt(r.total_price) },
    { key: 'asset_type',    label: 'ประเภท',      render: (r) => <StatusBadge value={r.asset_type} /> },
    { key: 'classified_by', label: 'จัดโดย' },
  ]

  return (
    /* ห่อทั้งหน้าด้วย flex column เต็มความสูง — ไม่ให้ page scroll */
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>

      {/* Floating viewer (popup on double-click) */}
      {showViewer && imageUrl && (
        <ImageViewer url={imageUrl} isPdf={isPdf} onClose={() => setShowViewer(false)} />
      )}

      {/* Nav + actions — fixed บนสุด ไม่ scroll หายไป */}
      <div className="flex items-center gap-3 mb-4 flex-wrap flex-shrink-0">
        <Btn variant="secondary" onClick={() => navigate(-1)}>← กลับ</Btn>
        <h2 className="text-xl font-semibold text-gray-800">Invoice Detail</h2>
        <div className="ml-auto flex gap-2 flex-wrap">
          {!editing && invoice.status !== 'verified' && (
            <Btn variant="secondary" onClick={handleReprocess} disabled={reprocessing}>
              {reprocessing ? 'กำลังส่ง…' : '↺ Re-run OCR'}
            </Btn>
          )}
          {!editing && invoice.status !== 'verified' && (
            <Btn onClick={startEdit}>✏ แก้ไข</Btn>
          )}
          {editing && (
            <>
              <Btn variant="secondary" onClick={cancelEdit} disabled={saving}>ยกเลิก</Btn>
              <Btn onClick={handleSave} disabled={saving}>
                {saving ? 'กำลังบันทึก…' : 'บันทึก'}
              </Btn>
            </>
          )}
        </div>
      </div>

      {/* 2-column layout — flex row, each column อิสระ */}
      <div style={{ display: 'flex', flex: 1, gap: '1.25rem', minHeight: 0 }}>

        {/* ── LEFT: scrollable column ── */}
        <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>

        {/* Invoice info card */}
        <div className="bg-white rounded-lg shadow p-5 text-sm space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-gray-400 mb-0.5">สถานะ</p>
              <StatusBadge value={invoice.status} />
            </div>
            {invoice.verified_at && (
              <div className="text-right">
                <p className="text-xs text-gray-400 mb-0.5">ยืนยันเมื่อ</p>
                <p className="text-xs text-green-700 font-medium">
                  {new Date(invoice.verified_at).toLocaleString('th-TH')}
                </p>
              </div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <Field label="ชื่อร้าน / ผู้ขาย" field="vendor_name" />
            <Field label="เลขที่ใบกำกับ (ในเอกสาร)" field="invoice_doc_no" mono />
            <Field label="เลขผู้เสียภาษีผู้ขาย" field="vendor_tax_id" mono />
            <Field label="วันที่ในใบกำกับ" field="invoice_date" />
          </div>

          <div className="text-xs text-gray-400">
            อัปโหลดเมื่อ {new Date(invoice.created_at).toLocaleString('th-TH')}
          </div>

          <div className="border-t pt-4">
            {editing ? (
              <div className="grid grid-cols-3 gap-3 text-center">
                {[
                  { label: 'ก่อน VAT', field: 'total_before_vat' },
                  { label: 'VAT (จากใบกำกับ)', field: 'vat_amount' },
                  { label: 'รวมสุทธิ', field: 'total_amount' },
                ].map(({ label, field }) => (
                  <div key={field}>
                    <p className="text-xs text-gray-400 mb-1">{label}</p>
                    <input
                      type="number"
                      step="0.01"
                      value={form[field] ?? 0}
                      onChange={(e) => handleField(field, e.target.value)}
                      className="border border-gray-300 rounded px-2 py-1 w-full text-sm text-center focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-3 gap-4 text-center">
                <div>
                  <p className="text-xs text-gray-400 mb-1">ก่อน VAT (ยืนยันแล้ว)</p>
                  {invoice.status === 'verified' ? (
                    <p className="text-lg font-semibold">{fmt(invoice.total_before_vat)}</p>
                  ) : (
                    <p className="text-lg font-semibold text-gray-300">รอยืนยัน</p>
                  )}
                </div>
                <div>
                  <p className="text-xs text-gray-400 mb-1">VAT (ยืนยันแล้ว)</p>
                  {invoice.status === 'verified' ? (
                    <>
                      <p className="text-lg font-semibold">{fmt(invoice.vat_amount)}</p>
                      <span className={`inline-block mt-1 text-xs px-2 py-0.5 rounded-full font-medium ${
                        invoice.vat_math_ok ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                      }`}>
                        {invoice.vat_math_ok ? '✓ สอดคล้อง 7%' : '⚠ ไม่สอดคล้อง 7%'}
                      </span>
                    </>
                  ) : (
                    <p className="text-lg font-semibold text-gray-300">รอยืนยัน</p>
                  )}
                </div>
                <div>
                  <p className="text-xs text-gray-400 mb-1">รวมสุทธิ (ยืนยันแล้ว)</p>
                  {invoice.status === 'verified' ? (
                    <p className="text-lg font-bold text-gray-800">{fmt(invoice.total_amount)}</p>
                  ) : (
                    <p className="text-lg font-bold text-gray-300">รอยืนยัน</p>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Line items */}
        <div>
          <h3 className="font-semibold text-gray-700 mb-3">Line Items ({items.length})</h3>
          <Table columns={itemCols} data={items} />
        </div>

        {invoice.status !== 'verified' && (
          <VerificationWizard
            invoice={invoice}
            items={items}
            onVerified={() => {
              loadInvoice()
              loadItems()
            }}
          />
        )}

        </div>{/* ── end LEFT column ── */}

        {/* ── RIGHT: image column — fixed height เพื่อให้ applyFitScale ทำงานถูก ── */}
        <div style={{ width: '50%', flexShrink: 0, height: 'calc(100vh - 200px)', alignSelf: 'flex-start' }} className="rounded-lg shadow overflow-hidden">
          {imageUrl ? (
            <ImageInteractive
              url={imageUrl}
              isPdf={isPdf}
              height="100%"
              onDoubleClick={() => setShowViewer(true)}
            />
          ) : (
            <div className="bg-white flex flex-col items-center justify-center text-gray-400 text-sm h-full gap-2 p-4">
              {imageError ? (
                <>
                  <span className="text-red-500 font-medium">โหลดรูปไม่ได้</span>
                  <span className="text-xs text-red-400 text-center break-all">{imageError}</span>
                </>
              ) : filePath === '' ? 'ไม่มีไฟล์ต้นฉบับ' : 'กำลังโหลดรูปภาพ…'}
            </div>
          )}
        </div>

      </div>{/* end 2-column */}
    </div>/* end page wrapper */
  )
}
