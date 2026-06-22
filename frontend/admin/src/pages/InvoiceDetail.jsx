import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { Btn, StatusBadge } from '../components/ui'
import VerificationWizard from './VerificationWizard'

function VendorVerifyModal({ vendor, ocrName, ocrAddress, onClose }) {
  const [form, setForm] = useState({
    name:        vendor.name || ocrName || '',
    address:     vendor.address || ocrAddress || '',
    branch_code: vendor.branch_code || '',
    branch_name: vendor.branch_name || '',
    phone:       vendor.phone || '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError]   = useState('')

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }))

  const submit = async (e) => {
    e.preventDefault()
    if (!form.name.trim()) { setError('กรุณาระบุชื่อผู้ขาย'); return }
    setSaving(true); setError('')
    try {
      await api.post(`/vendors/${vendor.id}/verify`, form)
      onClose(true)
    } catch (err) {
      setError(err.response?.data?.error || err.message)
    } finally {
      setSaving(false)
    }
  }

  const inputCls = 'w-full border border-gray-300 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-400'

  return (
    <Modal title="ยืนยันข้อมูลผู้ขาย" devLabel="P-05-M1 VendorVerify" onClose={() => onClose(false)}>
      <p className="text-xs text-gray-500 mb-1">
        เลขผู้เสียภาษี <span className="font-mono font-semibold text-gray-800">{vendor.tax_id}</span>
        {!vendor.verified && <span className="ml-2 text-amber-600">(ยังไม่ยืนยัน)</span>}
      </p>
      {!vendor.verified && (ocrName || ocrAddress) && (
        <div className="bg-blue-50 border border-blue-200 rounded p-2 text-xs text-blue-700 mb-3">
          <p className="font-semibold mb-0.5">ข้อมูลที่ OCR อ่านได้จากเอกสาร:</p>
          {ocrName && <p>ชื่อ: {ocrName}</p>}
          {ocrAddress && <p>ที่อยู่: {ocrAddress}</p>}
        </div>
      )}
      <form onSubmit={submit} className="space-y-3 mt-2">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">ชื่อบริษัท/ร้านค้า <span className="text-red-500">*</span></label>
          <input value={form.name} onChange={set('name')} className={inputCls} />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">รหัสสาขา</label>
            <input value={form.branch_code} onChange={set('branch_code')} className={inputCls} />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">ชื่อสาขา</label>
            <input value={form.branch_name} onChange={set('branch_name')} className={inputCls} />
          </div>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">เบอร์โทร</label>
          <input value={form.phone} onChange={set('phone')} className={inputCls} />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">ที่อยู่</label>
          <textarea value={form.address} onChange={set('address')} rows={3} className={inputCls} />
        </div>
        {error && <p className="text-red-500 text-xs">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Btn variant="secondary" onClick={() => onClose(false)}>ยกเลิก</Btn>
          <Btn type="submit" disabled={saving}>{saving ? 'กำลังบันทึก…' : '✓ ยืนยันและบันทึก'}</Btn>
        </div>
      </form>
    </Modal>
  )
}

/* ─────────────────────────────────────────────
   Shared interactive image core
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

  useEffect(() => { if (isPdf) applyFitScale() }, [isPdf])

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
          <button className={btnCls} onClick={onClose} style={{ background: '#7f1d1d', marginLeft: 'auto' }}>✕ ปิด</button>
        )}
        {onDoubleClick && (
          <span style={{ color: '#475569', fontSize: 11, marginLeft: 'auto' }}>ดับเบิ้ลคลิกเพื่อขยาย</span>
        )}
      </div>
      <div
        ref={containerRef}
        onMouseDown={onMouseDown}
        onDoubleClick={onDoubleClick}
        style={{
          overflow: 'hidden', flex: 1,
          cursor: panning ? 'grabbing' : 'grab',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: '#f1f5f9', userSelect: 'none',
        }}
      >
        {isPdf ? (
          <iframe
            src={url} title="invoice"
            style={{
              width: `${Math.round(680 * scale)}px`, height: `${Math.round(880 * scale)}px`,
              transform: `translate(${translate.x}px, ${translate.y}px) rotate(${rotate}deg)`,
              transformOrigin: 'center', transition: panning ? 'none' : 'transform 0.1s',
              border: 'none', pointerEvents: 'none', flexShrink: 0,
            }}
          />
        ) : (
          <img
            ref={imgRef} src={url} alt="invoice" draggable={false} onLoad={onImgLoad}
            style={{
              maxWidth: 'none', transform, transformOrigin: 'center',
              transition: panning ? 'none' : 'transform 0.1s',
              display: 'block', pointerEvents: 'none',
            }}
          />
        )}
      </div>
    </div>
  )
}

/* ─────────────────────────────────────────────
   Floating popup window
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
        position: 'fixed', top: pos.y, left: pos.x, zIndex: 9999,
        width: 800, maxWidth: '96vw', borderRadius: 10,
        boxShadow: '0 16px 60px rgba(0,0,0,0.45)', overflow: 'hidden',
        userSelect: dragging ? 'none' : 'auto',
      }}
    >
      <div
        onMouseDown={onHeaderMouseDown}
        style={{
          cursor: dragging ? 'grabbing' : 'grab', background: '#1e293b',
          padding: '9px 14px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        }}
      >
        <span style={{ color: '#e2e8f0', fontSize: 13, fontWeight: 600 }}>🖼 รูปต้นฉบับ — drag header เพื่อย้าย</span>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 10, fontFamily: 'monospace', background: 'rgba(255,255,255,0.15)', color: '#e2e8f0', padding: '2px 6px', borderRadius: 4 }}>P-05-M2 ImageViewer</span>
          <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#94a3b8', fontSize: 20, cursor: 'pointer', lineHeight: 1, padding: '0 4px' }}>✕</button>
        </div>
      </div>
      <ImageInteractive url={url} isPdf={isPdf} height="calc(85vh - 60px)" onClose={onClose} />
    </div>
  )
}

/* ─────────────────────────────────────────────
   Invoice Detail Page
───────────────────────────────────────────── */
const DOC_TYPE_LABELS = {
  tax_invoice: 'ใบกำกับภาษี',
  receipt: 'ใบเสร็จรับเงิน',
  invoice_billing: 'ใบแจ้งหนี้/ใบวางบิล',
  delivery_order: 'ใบส่งสินค้า/ใบรับของ',
  unknown: 'ไม่ทราบประเภท',
}

const INVALID_REASON_LABELS = {
  buyer_tax_id_mismatch:     'เลขผู้เสียภาษีผู้ซื้อไม่ตรงกับบริษัทของเรา — ภาษีซื้อต้องห้าม (ม.82/5)',
  buyer_branch_code_mismatch:'รหัสสาขาผู้ซื้อไม่ตรงกับสาขาของเรา — ภาษีซื้อต้องห้าม (ม.82/5)',
  buyer_name_mismatch:       'ชื่อผู้ซื้อไม่ตรงกับชื่อบริษัทของเรา — ภาษีซื้อต้องห้าม (ม.82/5)',
  late_invoice_vat_unclaimed:'ใบกำกับภาษีออกเกิน 3 เดือน — ไม่สามารถนำมาใช้เป็นภาษีซื้อใน ภพ.30 ได้',
}

export default function InvoiceDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [invoice, setInvoice]       = useState(null)
  const [items, setItems]           = useState([])
  const [tenant, setTenant]         = useState(null)
  const [imageUrl, setImageUrl]     = useState(null)
  const [filePath, setFilePath]     = useState('')
  const [imageError, setImageError] = useState(null)
  const [editing, setEditing]       = useState(false)
  const [form, setForm]             = useState({})
  const [saving, setSaving]         = useState(false)
  const [reprocessing, setReprocessing] = useState(false)
  const [showViewer, setShowViewer] = useState(false)

  const [vendor, setVendor] = useState(null)
  const [showVendorModal, setShowVendorModal] = useState(false)

  const loadInvoice = () => api.get(`/invoices/${id}`).then((r) => setInvoice(r.data.data))
  const loadItems   = () => api.get(`/invoices/${id}/items`).then((r) => setItems(r.data.data ?? []))

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
          try { const t = await err.response.data.text(); const p = JSON.parse(t); if (p.error) msg = p.error } catch (_) {}
        }
        setImageError(`${err.response?.status ?? ''} ${msg}`.trim())
      })
    return () => { if (blobUrl) URL.revokeObjectURL(blobUrl) }
  }, [id])

  // Fetch tenant for buyer match indicator
  useEffect(() => {
    if (!invoice?.tenant_id) return
    api.get(`/tenants/${invoice.tenant_id}`).then((r) => setTenant(r.data.data ?? r.data)).catch(() => {})
  }, [invoice?.tenant_id])

  // Fetch vendor info when invoice loads
  useEffect(() => {
    if (!invoice?.vendor_id) return
    api.get(`/vendors/${invoice.vendor_id}`).then((r) => setVendor(r.data.data)).catch(() => {})
  }, [invoice?.vendor_id])

  // Auto-refresh while pending
  useEffect(() => {
    if (invoice?.status !== 'pending') return
    const timer = setInterval(() => { loadInvoice(); loadItems() }, 3000)
    return () => clearInterval(timer)
  }, [invoice?.status])

  // Sync form when invoice loads (but not during active edit)
  useEffect(() => {
    if (invoice && !editing) setForm({ ...invoice })
  }, [invoice])

  if (!invoice) return <p className="text-gray-400">กำลังโหลด…</p>

  const fmt = (n) => {
    const parts = Number(n ?? 0).toFixed(2).split('.')
    parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    return parts.join('.')
  }

  const isPdf = filePath.toLowerCase().endsWith('.pdf')

  // Buyer match indicator
  const buyerTaxNorm  = (invoice.buyer_tax_id  || '').replace(/[\s-]/g, '')
  const tenantTaxNorm = (tenant?.tax_id || '').replace(/[\s-]/g, '')
  const buyerMatch = buyerTaxNorm && tenantTaxNorm
    ? buyerTaxNorm === tenantTaxNorm
    : null  // null = unknown

  // ── Edit helpers ──────────────────────────────────────────────
  const startEdit  = () => { setForm({ ...invoice }); setEditing(true) }
  const cancelEdit = () => { setForm({ ...invoice }); setEditing(false) }

  // Cascade: changing total_before_vat recalcs vat + total; changing vat recalcs total
  const handleField = (field, val) => {
    setForm((f) => {
      const next = { ...f, [field]: val }
      const rate = parseFloat(next.vat_rate) || 7
      if (field === 'total_before_vat') {
        const base = parseFloat(val) || 0
        const vat  = +(base * rate / 100).toFixed(2)
        next.vat_amount   = vat
        next.total_amount = +(base + vat).toFixed(2)
      } else if (field === 'vat_amount') {
        const base = parseFloat(next.total_before_vat) || 0
        next.total_amount = +(base + (parseFloat(val) || 0)).toFixed(2)
      }
      return next
    })
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const r = await api.put(`/invoices/${id}`, {
        doc_type:               form.doc_type || 'tax_invoice',
        vat_inclusive:          !!form.vat_inclusive,
        vat_rate:               parseFloat(form.vat_rate) || 7,
        vendor_name:            form.vendor_name || '',
        vendor_tax_id:          form.vendor_tax_id || '',
        vendor_address:         form.vendor_address || '',
        vendor_branch_code:     form.vendor_branch_code || '',
        buyer_name:             form.buyer_name || '',
        buyer_tax_id:           form.buyer_tax_id || '',
        buyer_address:          form.buyer_address || '',
        buyer_branch_code:      form.buyer_branch_code || '',
        invoice_doc_no:         form.invoice_doc_no || '',
        invoice_date:           form.invoice_date || '',
        vat_exempt_amount:      parseFloat(form.vat_exempt_amount) || 0,
        vat_inclusive_subtotal: parseFloat(form.vat_inclusive_subtotal) || 0,
        discount_amount:        parseFloat(form.discount_amount) || 0,
        total_before_vat:       parseFloat(form.total_before_vat) || 0,
        vat_amount:             parseFloat(form.vat_amount) || 0,
        total_amount:           parseFloat(form.total_amount) || 0,
      })
      setInvoice(r.data.data)
      setEditing(false)
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

  // ── Field components ─────────────────────────────────────────
  const inCls = 'border border-gray-300 rounded px-2 py-1 w-full text-sm focus:outline-none focus:ring-1 focus:ring-blue-500'

  const ViewVal = ({ val, mono }) => (
    <p className={`font-medium text-gray-800 text-sm ${mono ? 'font-mono' : ''}`}>{val || '—'}</p>
  )

  const F = ({ label, field, mono = false, textarea = false }) => (
    <div>
      <p className="text-xs text-gray-400 mb-0.5">{label}</p>
      {editing ? (
        textarea
          ? <textarea rows={2} value={form[field] ?? ''} onChange={(e) => handleField(field, e.target.value)} className={inCls + ' resize-y'} />
          : <input value={form[field] ?? ''} onChange={(e) => handleField(field, e.target.value)} className={inCls} />
      ) : (
        <ViewVal val={invoice[field]} mono={mono} />
      )}
    </div>
  )

  const NF = ({ label, field, cascade = true }) => (
    <div>
      <p className="text-xs text-gray-400 mb-0.5">{label}</p>
      {editing ? (
        <input
          type="number" step="0.01"
          value={form[field] ?? 0}
          onChange={(e) => cascade ? handleField(field, e.target.value) : setForm((f) => ({ ...f, [field]: e.target.value }))}
          className={inCls + ' text-right'}
        />
      ) : (
        <p className="font-medium text-gray-800 text-sm text-right font-mono">{fmt(invoice[field])}</p>
      )}
    </div>
  )

  // ── Item table columns ──────────────────────────────────────
  const itemCols = [
    { key: 'product_code', label: 'รหัส', render: (r) => r.product_code || '—' },
    { key: 'description',  label: 'รายการ' },
    { key: 'unit',         label: 'หน่วย', render: (r) => r.unit || '—' },
    { key: 'quantity',     label: 'จำนวน',      render: (r) => fmt(r.quantity) },
    { key: 'unit_price',   label: 'ราคา/หน่วย', render: (r) => fmt(r.unit_price) },
    { key: 'discount',     label: 'ส่วนลด',     render: (r) => r.discount ? fmt(r.discount) : '—' },
    { key: 'total_price',  label: 'รวม',         render: (r) => fmt(r.total_price) },
    { key: 'asset_type',   label: 'ประเภท',      render: (r) => <StatusBadge value={r.asset_type} /> },
    { key: 'classified_by',label: 'จัดโดย' },
  ]

  // ── Buyer match badge ────────────────────────────────────────
  const BuyerMatchBadge = () => {
    if (buyerMatch === true)
      return <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-green-100 text-green-700 font-medium">✓ ตรงกับบริษัทของเรา</span>
    if (buyerMatch === false)
      return <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-yellow-100 text-yellow-700 font-medium">⚠ ไม่ตรงกับบริษัทของเรา</span>
    return <span className="text-xs text-gray-400">—</span>
  }

  const CardTitle = ({ children }) => (
    <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">{children}</h3>
  )

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>

      {showViewer && imageUrl && (
        <ImageViewer url={imageUrl} isPdf={isPdf} onClose={() => setShowViewer(false)} />
      )}

      {/* Nav + actions */}
      <div className="flex items-center gap-3 mb-4 flex-wrap flex-shrink-0">
        <Btn variant="secondary" onClick={() => navigate(-1)}>← กลับ</Btn>
        <h2 className="text-xl font-semibold text-gray-800">
          Invoice #{invoice.invoice_no}
          {invoice.invoice_doc_no && <span className="text-base font-normal text-gray-500 ml-2">({invoice.invoice_doc_no})</span>}
        </h2>
        <StatusBadge value={invoice.status} />
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
              <Btn onClick={handleSave} disabled={saving}>{saving ? 'กำลังบันทึก…' : 'บันทึก'}</Btn>
            </>
          )}
        </div>
      </div>

      {/* Invalid alert banner */}
      {invoice.status === 'invalid' && invoice.invalid_reason && (
        <div className="mb-3 flex items-start gap-2 bg-red-50 border border-red-300 rounded-lg px-4 py-3 text-sm text-red-800 flex-shrink-0">
          <span className="text-lg leading-none mt-0.5">⛔</span>
          <div>
            <p className="font-semibold mb-0.5">เอกสารไม่ถูกต้องตามกฎหมาย — ไม่สามารถใช้เป็นภาษีซื้อได้</p>
            <p className="text-red-700">{INVALID_REASON_LABELS[invoice.invalid_reason] || invoice.invalid_reason}</p>
            <p className="text-xs text-red-500 mt-1">กรุณาแก้ไขข้อมูลให้ถูกต้องแล้ว Re-run OCR หรือแก้ไขเอกสาร</p>
          </div>
        </div>
      )}

      {/* Late invoice warning (status = verified but has reason) */}
      {invoice.status !== 'invalid' && invoice.invalid_reason === 'late_invoice_vat_unclaimed' && (
        <div className="mb-3 flex items-start gap-2 bg-amber-50 border border-amber-300 rounded-lg px-4 py-3 text-sm text-amber-800 flex-shrink-0">
          <span className="text-lg leading-none mt-0.5">⚠️</span>
          <div>
            <p className="font-semibold mb-0.5">บิลข้ามเดือนเกิน 3 เดือน</p>
            <p className="text-amber-700">ไม่สามารถนำภาษีซื้อในบิลนี้มายื่น ภพ.30 ได้ — บันทึกเป็นค่าใช้จ่ายได้ตามปกติ</p>
          </div>
        </div>
      )}

      {/* 2-column layout */}
      <div style={{ display: 'flex', flex: 1, gap: '1.25rem', minHeight: 0 }}>

        {/* ── LEFT: scrollable ── */}
        <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '1rem' }}>

          {/* ── Document Info ── */}
          <div className="bg-white rounded-lg shadow p-4 text-sm">
            <CardTitle>เอกสาร</CardTitle>
            <div className="grid grid-cols-2 gap-3">
              {/* doc_type */}
              <div>
                <p className="text-xs text-gray-400 mb-1">ประเภทเอกสาร</p>
                {editing ? (
                  <select value={form.doc_type || 'tax_invoice'} onChange={(e) => handleField('doc_type', e.target.value)} className={inCls}>
                    <option value="tax_invoice">ใบกำกับภาษี</option>
                    <option value="receipt">ใบเสร็จรับเงิน</option>
                    <option value="invoice_billing">ใบแจ้งหนี้/ใบวางบิล</option>
                    <option value="delivery_order">ใบส่งสินค้า/ใบรับของ</option>
                    <option value="unknown">ไม่ทราบประเภท</option>
                  </select>
                ) : (
                  <span className="inline-block px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-700">
                    {DOC_TYPE_LABELS[invoice.doc_type] || invoice.doc_type || '—'}
                  </span>
                )}
              </div>
              {/* vat_inclusive */}
              <div>
                <p className="text-xs text-gray-400 mb-1">ราคาในบิล</p>
                {editing ? (
                  <label className="flex items-center gap-2 text-sm cursor-pointer">
                    <input type="checkbox" checked={!!form.vat_inclusive} onChange={(e) => handleField('vat_inclusive', e.target.checked)} className="w-4 h-4" />
                    ราคารวม VAT แล้ว
                  </label>
                ) : (
                  invoice.vat_inclusive
                    ? <span className="inline-block px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-700">รวม VAT แล้ว</span>
                    : <span className="inline-block px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600">ยังไม่รวม VAT</span>
                )}
              </div>
              <F label="เลขที่เอกสาร" field="invoice_doc_no" mono />
              <F label="วันที่ในเอกสาร" field="invoice_date" />
              {/* vat_rate */}
              <div>
                <p className="text-xs text-gray-400 mb-0.5">อัตรา VAT (%)</p>
                {editing ? (
                  <input type="number" step="0.01" value={form.vat_rate ?? 7} onChange={(e) => handleField('vat_rate', e.target.value)} className={inCls} />
                ) : (
                  <p className="font-medium text-gray-800 text-sm">{invoice.vat_rate ?? 7}%</p>
                )}
              </div>
              <div>
                <p className="text-xs text-gray-400 mb-0.5">อัปโหลดเมื่อ</p>
                <p className="text-sm text-gray-500">{new Date(invoice.created_at).toLocaleString('th-TH')}</p>
              </div>
            </div>
          </div>

          {/* ── Vendor verification banner ── */}
          {vendor && !vendor.verified && (
            <div className="bg-amber-50 border border-amber-300 rounded-lg p-4 flex items-start gap-3">
              <span className="text-amber-500 text-xl">⚠️</span>
              <div className="flex-1">
                <p className="font-semibold text-amber-800 text-sm">ยังไม่ยืนยันข้อมูลผู้ขาย</p>
                <p className="text-xs text-amber-700 mt-0.5">
                  เลขผู้เสียภาษี <span className="font-mono font-bold">{vendor.tax_id}</span> — ชื่อและที่อยู่ถูกอ่านจาก OCR อัตโนมัติ กรุณาตรวจสอบและยืนยันให้ถูกต้อง
                </p>
              </div>
              <button
                onClick={() => setShowVendorModal(true)}
                className="shrink-0 bg-amber-500 hover:bg-amber-600 text-white text-xs font-semibold px-3 py-1.5 rounded">
                ยืนยันข้อมูล
              </button>
            </div>
          )}
          {vendor && vendor.verified && (
            <div className="bg-green-50 border border-green-200 rounded-lg px-4 py-2 flex items-center gap-2 text-xs text-green-700">
              <span>✅</span>
              <span>ผู้ขายยืนยันแล้ว — <span className="font-semibold">{vendor.name}</span></span>
            </div>
          )}

          {/* ── Vendor ── */}
          <div className="bg-white rounded-lg shadow p-4 text-sm">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide">ผู้ขาย (Vendor)</h3>
              {vendor && (
                <button onClick={() => setShowVendorModal(true)}
                  className="text-xs text-indigo-600 hover:underline">
                  {vendor.verified ? 'ดู/แก้ไขข้อมูลในทะเบียน' : 'ยืนยันข้อมูลผู้ขาย'}
                </button>
              )}
            </div>
            <div className="grid grid-cols-2 gap-3">
              <F label="ชื่อบริษัท/ร้านค้า" field="vendor_name" />
              <F label="เลขผู้เสียภาษี (13 หลัก)" field="vendor_tax_id" mono />
              <F label="รหัสสาขา" field="vendor_branch_code" mono />
              <div className="col-span-2">
                <F label="ที่อยู่" field="vendor_address" textarea />
              </div>
            </div>
          </div>

          {/* ── Vendor Verify Modal ── */}
          {showVendorModal && vendor && (
            <VendorVerifyModal
              vendor={vendor}
              ocrName={invoice.vendor_name}
              ocrAddress={invoice.vendor_address}
              onClose={(updated) => {
                setShowVendorModal(false)
                if (updated) {
                  api.get(`/vendors/${vendor.id}`).then((r) => setVendor(r.data.data))
                }
              }}
            />
          )}

          {/* ── Buyer ── */}
          <div className="bg-white rounded-lg shadow p-4 text-sm">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide">ผู้ซื้อ (Buyer)</h3>
              <BuyerMatchBadge />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <F label="ชื่อบริษัท/ลูกค้า" field="buyer_name" />
              <F label="เลขผู้เสียภาษี (13 หลัก)" field="buyer_tax_id" mono />
              <F label="รหัสสาขา" field="buyer_branch_code" mono />
              <div className="col-span-2">
                <F label="ที่อยู่" field="buyer_address" textarea />
              </div>
            </div>
          </div>

          {/* ── Financial Summary ── */}
          <div className="bg-white rounded-lg shadow p-4 text-sm">
            <CardTitle>ยอดเงิน</CardTitle>

            {/* Reference values (always visible) */}
            <div className="grid grid-cols-3 gap-3 mb-4 pb-4 border-b">
              <NF label="มูลค่าที่ยกเว้นภาษี" field="vat_exempt_amount" cascade={false} />
              <NF label="มูลค่าที่มีภาษี (VAT-inc)" field="vat_inclusive_subtotal" cascade={false} />
              <NF label="ส่วนลดรวม" field="discount_amount" cascade={false} />
            </div>

            {/* Confirmed amounts */}
            {editing ? (
              <div className="grid grid-cols-3 gap-3">
                <NF label="ก่อน VAT" field="total_before_vat" />
                <NF label="VAT" field="vat_amount" />
                <NF label="รวมสุทธิ" field="total_amount" cascade={false} />
              </div>
            ) : (
              <div className="grid grid-cols-3 gap-4 text-center">
                <div>
                  <p className="text-xs text-gray-400 mb-1">ก่อน VAT</p>
                  {invoice.status === 'verified'
                    ? <p className="text-lg font-semibold">{fmt(invoice.total_before_vat)}</p>
                    : <p className="text-lg font-semibold text-gray-300">รอยืนยัน</p>}
                </div>
                <div>
                  <p className="text-xs text-gray-400 mb-1">VAT</p>
                  {invoice.status === 'verified' ? (
                    <>
                      <p className="text-lg font-semibold">{fmt(invoice.vat_amount)}</p>
                      <span className={`inline-block mt-1 text-xs px-2 py-0.5 rounded-full font-medium ${invoice.vat_math_ok ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
                        {invoice.vat_math_ok ? `✓ สอดคล้อง ${invoice.vat_rate}%` : `⚠ ไม่สอดคล้อง ${invoice.vat_rate}%`}
                      </span>
                    </>
                  ) : <p className="text-lg font-semibold text-gray-300">รอยืนยัน</p>}
                </div>
                <div>
                  <p className="text-xs text-gray-400 mb-1">รวมสุทธิ</p>
                  {invoice.status === 'verified'
                    ? <p className="text-xl font-bold text-gray-800">{fmt(invoice.total_amount)}</p>
                    : <p className="text-xl font-bold text-gray-300">รอยืนยัน</p>}
                </div>
              </div>
            )}
            {editing && (
              <p className="text-xs text-gray-400 mt-2">* เปลี่ยน "ก่อน VAT" จะคำนวณ VAT + รวมสุทธิ อัตโนมัติ (สามารถแก้ได้)</p>
            )}
            {invoice.verified_at && (
              <p className="text-xs text-green-600 mt-3">✓ ยืนยันเมื่อ {new Date(invoice.verified_at).toLocaleString('th-TH')}</p>
            )}
          </div>

          {/* ── Line Items ── */}
          <div>
            <h3 className="font-semibold text-gray-700 mb-3">Line Items ({items.length})</h3>
            <Table columns={itemCols} data={items} />
          </div>

          {/* ── Verification Wizard ── */}
          {invoice.status !== 'verified' && (
            <VerificationWizard
              invoice={invoice}
              items={items}
              onVerified={() => { loadInvoice(); loadItems() }}
            />
          )}

        </div>{/* end LEFT */}

        {/* ── RIGHT: image viewer ── */}
        <div style={{ width: '50%', flexShrink: 0, height: 'calc(100vh - 200px)', alignSelf: 'flex-start' }} className="rounded-lg shadow overflow-hidden">
          {imageUrl ? (
            <ImageInteractive url={imageUrl} isPdf={isPdf} height="100%" onDoubleClick={() => setShowViewer(true)} />
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
    </div>
  )
}
