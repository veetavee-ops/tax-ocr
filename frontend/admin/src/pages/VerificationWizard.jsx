import { useState, useMemo } from 'react'
import api from '../api/client'
import { Btn } from '../components/ui'

const r2 = n => Math.round(n * 100) / 100
const fmt = n => Number(n ?? 0).toLocaleString('th-TH', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

function SectionHeader({ title, state }) {
  const cfg = {
    ok:       { icon: '✓', cls: 'text-green-600' },
    conflict: { icon: '✗', cls: 'text-red-500' },
    resolved: { icon: '✓', cls: 'text-amber-500' },
  }[state] ?? { icon: '○', cls: 'text-gray-400' }

  return (
    <div className={`flex items-center gap-2 mb-3 ${cfg.cls}`}>
      <span className="font-bold text-base">{cfg.icon}</span>
      <span className="font-semibold text-gray-700 text-sm">{title}</span>
      {state === 'conflict'  && <span className="text-xs text-red-400 ml-1">— กรุณาเลือกค่าที่ถูกต้อง</span>}
      {state === 'resolved'  && <span className="text-xs text-amber-500 ml-1">— แก้ไขแล้ว</span>}
    </div>
  )
}

function ChoiceBtn({ label, value, chosen, onChoose }) {
  const active = Math.abs(value - chosen) < 0.005
  return (
    <button
      onClick={() => onChoose(value)}
      className={`flex items-center justify-between w-full px-3 py-2 rounded border text-sm transition-colors ${
        active
          ? 'border-blue-500 bg-blue-50 text-blue-700 font-semibold'
          : 'border-gray-200 bg-white text-gray-600 hover:border-blue-300'
      }`}
    >
      <span className="text-xs">{label}</span>
      <span className="font-mono font-semibold">{fmt(value)} บาท</span>
    </button>
  )
}

export default function VerificationWizard({ invoice, items, onVerified }) {
  // ── user choices (start with OCR values) ──
  const [itemChoices, setItemChoices]   = useState(() => Object.fromEntries(items.map(i => [i.id, i.total_price])))
  const [itemResolved, setItemResolved] = useState(() => new Set())
  const [itemEditing, setItemEditing]   = useState(() => ({})) // { id: { qty, unitPrice, totalPrice } }
  const [chosenVAT,   setChosenVAT]    = useState(invoice.vat_amount)
  const [vatResolved, setVatResolved]  = useState(false)
  const [chosenTotal, setChosenTotal]  = useState(invoice.total_amount)
  const [totalResolved, setTotalResolved] = useState(false)
  const [vatCustom,   setVatCustom]    = useState('')
  const [totalCustom, setTotalCustom]  = useState('')
  const [saving, setSaving]            = useState(false)

  // ── computed checks ──────────────────────────────────────────
  const itemChecks = useMemo(() => items.map(item => {
    const computed    = r2(item.quantity * item.unit_price)
    const canCompute  = item.quantity > 0 && item.unit_price > 0
    const hasConflict = canCompute && Math.abs(computed - item.total_price) >= 0.01
    const missingData = !canCompute && item.total_price > 0
    const chosen      = itemChoices[item.id] ?? item.total_price
    return { ...item, computed, chosen, hasConflict, missingData, canCompute }
  }), [items, itemChoices])

  const itemsSum       = useMemo(() => r2(itemChecks.reduce((s, i) => s + (itemChoices[i.id] ?? i.total_price), 0)), [itemChecks, itemChoices])
  const subtotalConflict = items.length > 0
                           && invoice.total_before_vat > 0
                           && Math.abs(itemsSum - invoice.total_before_vat) >= 0.01

  const vatCalc       = r2(invoice.total_before_vat * 0.07)
  const vatConflict   = invoice.total_before_vat > 0 && Math.abs(vatCalc - invoice.vat_amount) >= 0.01

  const totalCalc     = r2(invoice.total_before_vat + chosenVAT)
  const totalConflict = Math.abs(totalCalc - invoice.total_amount) >= 0.01

  // ── resolution status ─────────────────────────────────────────
  const conflictItems    = itemChecks.filter(i => i.hasConflict)
  const allItemsResolved = conflictItems.every(i => itemResolved.has(i.id))
  const vatOKOrResolved  = !vatConflict || vatResolved
  const totalOKOrResolved = !totalConflict || totalResolved

  // Only block on things the user can explicitly resolve (item rows + VAT choice).
  // subtotalConflict and totalConflict are shown as warnings but do not block.
  const canVerify = allItemsResolved && vatOKOrResolved

  // ── helpers ───────────────────────────────────────────────────
  const resolveItem = (id, value) => {
    setItemChoices(p  => ({ ...p, [id]: value }))
    setItemResolved(p => new Set([...p, id]))
  }

  const resolveVAT = value => {
    setChosenVAT(value)
    setVatResolved(true)
    setVatCustom('')
    // When user confirms VAT, auto-compute the implied total (before_vat + chosen_vat).
    // If that differs from OCR total, pre-select it so Level 3 doesn't block canVerify.
    // User can still override Level 3 manually if they disagree.
    const impliedTotal = r2(invoice.total_before_vat + value)
    setChosenTotal(impliedTotal)
    setTotalResolved(Math.abs(impliedTotal - invoice.total_amount) >= 0.01)
  }

  const resolveTotal = value => {
    setChosenTotal(value)
    setTotalResolved(true)
    setTotalCustom('')
  }

  const applyCustomVAT   = () => { const v = parseFloat(vatCustom);   if (!isNaN(v)) resolveVAT(r2(v)) }
  const applyCustomTotal = () => { const v = parseFloat(totalCustom); if (!isNaN(v)) resolveTotal(r2(v)) }

  // ── save & verify ─────────────────────────────────────────────
  const handleSave = async () => {
    setSaving(true)
    try {
      // 1. Save item corrections
      for (const item of itemChecks) {
        const chosen = itemChoices[item.id] ?? item.total_price
        if (Math.abs(chosen - item.total_price) >= 0.005) {
          await api.put(`/invoice-items/${item.id}`, {
            quantity:    item.quantity,
            unit_price:  item.unit_price,
            total_price: chosen,
          })
        }
      }

      // 2. Verify + apply confirmed values atomically in one call
      await api.post(`/invoices/${invoice.id}/verify`, {
        total_before_vat: invoice.total_before_vat,
        vat_amount:       chosenVAT,
        total_amount:     chosenTotal,
      })
      // Reload from DB — header will now show the values we just saved
      onVerified()
    } catch (e) {
      alert('เกิดข้อผิดพลาด: ' + (e.response?.data?.error || e.message))
    } finally {
      setSaving(false)
    }
  }

  // ── item state helper ─────────────────────────────────────────
  const sectionState = (hasConflict, resolved) =>
    !hasConflict ? 'ok' : resolved ? 'resolved' : 'conflict'

  const level1State = (!conflictItems.length && !subtotalConflict)
    ? 'ok'
    : allItemsResolved && !subtotalConflict ? 'resolved' : 'conflict'

  return (
    <div className="bg-white rounded-lg shadow p-5 text-sm">
      <h3 className="font-semibold text-gray-700 mb-5 border-b pb-3">ตรวจสอบความถูกต้อง</h3>

      {/* ── Level 1: Line Items ── */}
      <section className="mb-6">
        <SectionHeader title="Level 1 — รายการสินค้า" state={level1State} />
        <div className="ml-5">
          {items.length === 0 ? (
            <p className="text-xs text-gray-400">ไม่มีรายการสินค้า</p>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full text-xs border border-gray-100 rounded">
                  <thead>
                    <tr className="bg-gray-50 text-gray-500">
                      <th className="px-2 py-1.5 text-left font-medium">รายการ</th>
                      <th className="px-2 py-1.5 text-right font-medium">จำนวน</th>
                      <th className="px-2 py-1.5 text-right font-medium">ราคา/หน่วย</th>
                      <th className="px-2 py-1.5 text-right font-medium">รวม (OCR)</th>
                      <th className="px-2 py-1.5 text-right font-medium">รวม (คำนวณ)</th>
                      <th className="px-2 py-1.5 text-center font-medium w-28">เลือก</th>
                    </tr>
                  </thead>
                  <tbody>
                    {itemChecks.map(item => {
                      const chosen  = itemChoices[item.id] ?? item.total_price
                      const editing = itemEditing[item.id]
                      const rowBg   = item.hasConflict ? 'bg-red-50' : item.missingData ? 'bg-yellow-50' : ''
                      return (
                        <tr key={item.id} className={`border-t ${rowBg}`}>
                          <td className="px-2 py-1.5 text-gray-800">{item.description || '—'}</td>
                          <td className="px-2 py-1.5 text-right font-mono">{item.quantity}</td>
                          <td className="px-2 py-1.5 text-right font-mono">{fmt(item.unit_price)}</td>
                          <td className={`px-2 py-1.5 text-right font-mono ${item.hasConflict ? 'text-red-500 font-semibold' : 'text-gray-700'}`}>
                            {fmt(item.total_price)}
                          </td>
                          <td className={`px-2 py-1.5 text-right font-mono ${item.hasConflict ? 'text-blue-600 font-semibold' : item.canCompute ? 'text-gray-400' : 'text-gray-300'}`}>
                            {item.canCompute ? fmt(item.computed) : '—'}
                          </td>
                          <td className="px-2 py-1.5 text-center">
                            {editing ? (
                              <div className="flex gap-1 items-center justify-center">
                                <input
                                  type="number" step="0.01"
                                  defaultValue={chosen}
                                  onBlur={e => {
                                    const v = parseFloat(e.target.value)
                                    if (!isNaN(v)) resolveItem(item.id, r2(v))
                                    setItemEditing(p => { const n = {...p}; delete n[item.id]; return n })
                                  }}
                                  onKeyDown={e => {
                                    if (e.key === 'Enter') e.target.blur()
                                    if (e.key === 'Escape') setItemEditing(p => { const n = {...p}; delete n[item.id]; return n })
                                  }}
                                  autoFocus
                                  className="border border-blue-400 rounded px-1.5 py-0.5 text-xs w-20 font-mono"
                                />
                              </div>
                            ) : item.hasConflict ? (
                              <div className="flex gap-1 justify-center flex-wrap">
                                <button
                                  onClick={() => resolveItem(item.id, item.total_price)}
                                  className={`px-2 py-0.5 rounded text-xs border transition-colors ${
                                    itemResolved.has(item.id) && Math.abs(chosen - item.total_price) < 0.005
                                      ? 'bg-red-100 border-red-400 text-red-700 font-semibold'
                                      : 'border-gray-300 hover:border-red-300 text-gray-500'
                                  }`}
                                >OCR</button>
                                <button
                                  onClick={() => resolveItem(item.id, item.computed)}
                                  className={`px-2 py-0.5 rounded text-xs border transition-colors ${
                                    itemResolved.has(item.id) && Math.abs(chosen - item.computed) < 0.005
                                      ? 'bg-blue-100 border-blue-400 text-blue-700 font-semibold'
                                      : 'border-gray-300 hover:border-blue-300 text-gray-500'
                                  }`}
                                >คำนวณ</button>
                                <button
                                  onClick={() => setItemEditing(p => ({...p, [item.id]: true}))}
                                  className="text-gray-300 hover:text-blue-400 text-xs"
                                  title="กรอกเอง"
                                >✎</button>
                              </div>
                            ) : item.missingData ? (
                              <button
                                onClick={() => setItemEditing(p => ({...p, [item.id]: true}))}
                                className="px-2 py-0.5 rounded text-xs border border-yellow-400 bg-yellow-50 text-yellow-700 hover:bg-yellow-100 transition-colors"
                              >✎ กรอกยอด</button>
                            ) : (
                              <div className="flex items-center gap-1 justify-center">
                                {itemResolved.has(item.id)
                                  ? <span className="text-amber-500 text-xs font-semibold">✓ {fmt(chosen)}</span>
                                  : <span className="text-green-500 font-bold">✓</span>
                                }
                                <button
                                  onClick={() => setItemEditing(p => ({...p, [item.id]: true}))}
                                  className="text-gray-300 hover:text-blue-400 text-xs ml-1"
                                  title="แก้ไขค่า"
                                >✎</button>
                              </div>
                            )}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>

              {/* Subtotal row */}
              <div className={`mt-2 px-3 py-2 rounded text-xs flex items-center gap-4 ${
                subtotalConflict ? 'bg-yellow-50 border border-yellow-200' : 'bg-green-50 border border-green-100'
              }`}>
                <span>ผลรวมรายการ: <strong className="font-mono">{fmt(itemsSum)}</strong></span>
                <span className="text-gray-400">vs</span>
                <span>ยอดก่อน VAT: <strong className="font-mono">{fmt(invoice.total_before_vat)}</strong></span>
                <span className={`ml-auto font-semibold ${subtotalConflict ? 'text-yellow-600' : 'text-green-600'}`}>
                  {subtotalConflict ? '⚠ ไม่ตรง' : '✓ ตรง'}
                </span>
              </div>
            </>
          )}
        </div>
      </section>

      {/* ── Level 2: VAT ── */}
      <section className="mb-6">
        <SectionHeader
          title="Level 2 — ภาษีมูลค่าเพิ่ม (VAT)"
          state={sectionState(vatConflict, vatResolved)}
        />
        <div className="ml-5">
          {vatConflict ? (
            <div className="space-y-2">
              <p className="text-xs text-gray-500 mb-3">
                VAT ใบกำกับ <strong className="text-red-500 font-mono">{fmt(invoice.vat_amount)}</strong> ≠ คำนวณ{' '}
                <strong className="text-blue-600 font-mono">{fmt(invoice.total_before_vat)} × 7% = {fmt(vatCalc)}</strong>
              </p>
              <ChoiceBtn label="ค่า VAT จาก OCR" value={invoice.vat_amount} chosen={chosenVAT} onChoose={resolveVAT} />
              <ChoiceBtn label={`คำนวณ 7% (${fmt(invoice.total_before_vat)} × 7%)`} value={vatCalc} chosen={chosenVAT} onChoose={resolveVAT} />
              <div className="flex items-center gap-2 pt-1">
                <span className="text-xs text-gray-400">กรอกเอง:</span>
                <input
                  type="number" step="0.01" value={vatCustom}
                  onChange={e => setVatCustom(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && applyCustomVAT()}
                  placeholder="0.00"
                  className="border border-gray-300 rounded px-2 py-1 text-xs w-28 font-mono"
                />
                <button onClick={applyCustomVAT} className="px-2 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded border border-gray-200">
                  ใช้ค่านี้
                </button>
              </div>
              {vatResolved && (
                <p className="text-xs text-amber-600 font-medium pt-1">✓ เลือกแล้ว: VAT = {fmt(chosenVAT)} บาท</p>
              )}
            </div>
          ) : (
            <p className="text-xs text-green-600">
              ✓ {fmt(invoice.total_before_vat)} × 7% = <strong className="font-mono">{fmt(vatCalc)}</strong> ตรงกับ OCR
            </p>
          )}
        </div>
      </section>

      {/* ── Level 3: Grand Total ── */}
      <section className="mb-6">
        <SectionHeader
          title="Level 3 — ยอดรวมสุทธิ"
          state={sectionState(totalConflict, totalResolved)}
        />
        <div className="ml-5">
          {totalConflict ? (
            <div className="space-y-2">
              <p className="text-xs text-gray-500 mb-3">
                คำนวณ <strong className="text-blue-600 font-mono">{fmt(invoice.total_before_vat)} + {fmt(chosenVAT)} = {fmt(totalCalc)}</strong>{' '}
                ≠ ยอดรวม OCR <strong className="text-red-500 font-mono">{fmt(invoice.total_amount)}</strong>
              </p>
              <ChoiceBtn label={`คำนวณ (${fmt(invoice.total_before_vat)} + ${fmt(chosenVAT)})`} value={totalCalc} chosen={chosenTotal} onChoose={resolveTotal} />
              <ChoiceBtn label="ยอดรวมจาก OCR" value={invoice.total_amount} chosen={chosenTotal} onChoose={resolveTotal} />
              <div className="flex items-center gap-2 pt-1">
                <span className="text-xs text-gray-400">กรอกเอง:</span>
                <input
                  type="number" step="0.01" value={totalCustom}
                  onChange={e => setTotalCustom(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && applyCustomTotal()}
                  placeholder="0.00"
                  className="border border-gray-300 rounded px-2 py-1 text-xs w-28 font-mono"
                />
                <button onClick={applyCustomTotal} className="px-2 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded border border-gray-200">
                  ใช้ค่านี้
                </button>
              </div>
              {totalResolved && (
                <p className="text-xs text-amber-600 font-medium pt-1">✓ เลือกแล้ว: ยอดรวม = {fmt(chosenTotal)} บาท</p>
              )}
            </div>
          ) : (
            <p className="text-xs text-green-600">
              ✓ {fmt(invoice.total_before_vat)} + {fmt(chosenVAT)} = <strong className="font-mono">{fmt(totalCalc)}</strong> ตรงกับ OCR
            </p>
          )}
        </div>
      </section>

      {/* ── Action ── */}
      <div className="border-t pt-4 flex items-center gap-4">
        {!canVerify && (
          <p className="text-xs text-yellow-600">⚠ แก้ไขรายการที่ไม่ตรงก่อนยืนยัน</p>
        )}
        <Btn variant="success" onClick={handleSave} disabled={!canVerify || saving} className="ml-auto">
          {saving ? 'กำลังบันทึก…' : '✓ บันทึกและยืนยัน'}
        </Btn>
      </div>
    </div>
  )
}
