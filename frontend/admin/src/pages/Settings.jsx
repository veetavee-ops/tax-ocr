import { useEffect, useRef, useState } from 'react'
import api from '../api/client'
import { PageHeader, Btn } from '../components/ui'

// ─── Reward Config ──────────────────────────────────────────────────────────

function RewardSection() {
  const [rewards, setRewards] = useState([])
  const [editing, setEditing] = useState({})
  const [msg, setMsg] = useState('')

  const load = () =>
    api.get('/reward/config').then((r) => {
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
    <div className="bg-white rounded-lg shadow p-6 max-w-lg mb-6">
      <h3 className="font-semibold text-gray-700 mb-4">Reward Config</h3>
      {rewards.length === 0
        ? <p className="text-sm text-gray-400">ยังไม่มีข้อมูล reward config</p>
        : rewards.map((r) => (
          <div key={r.id} className="flex items-center gap-3 mb-3">
            <span className="text-sm text-gray-600 w-56">{r.task_type}</span>
            <input
              type="number"
              value={editing[r.id] ?? r.amount}
              onChange={(e) => setEditing((p) => ({ ...p, [r.id]: e.target.value }))}
              className="border border-gray-300 rounded px-3 py-1.5 text-sm w-24"
            />
            <span className="text-sm text-gray-400">{r.currency}</span>
            <Btn onClick={() => save(r.id)}>บันทึก</Btn>
          </div>
        ))
      }
      {msg && <p className={`text-sm mt-3 ${msg === 'บันทึกสำเร็จ' ? 'text-green-600' : 'text-red-500'}`}>{msg}</p>}
    </div>
  )
}

// ─── OCR Engine Config ───────────────────────────────────────────────────────

const PROVIDER_LABEL = { openai: 'OpenAI GPT-4o-mini', gcv: 'Google Cloud Vision' }
const PROVIDER_DESC  = {
  openai: 'ดึงโครงสร้างใบกำกับภาษี (Invoice Structure Extraction)',
  gcv:    'อ่านตัวอักษรจากภาพ (Document Text Detection)',
}

function OCREngineSection() {
  const [configs, setConfigs] = useState([])
  const [keys, setKeys] = useState({})     // { provider: keyValue }
  const [enabled, setEnabled] = useState({})
  const [msgs, setMsgs] = useState({})

  const load = () =>
    api.get('/ocr/config').then((r) => {
      const data = r.data.data ?? []
      setConfigs(data)
      setKeys(Object.fromEntries(data.map((d) => [d.provider, ''])))
      setEnabled(Object.fromEntries(data.map((d) => [d.provider, d.enabled])))
    })

  useEffect(() => { load() }, [])

  const save = async (provider) => {
    setMsgs((p) => ({ ...p, [provider]: '' }))
    try {
      await api.put(`/ocr/config/${provider}`, {
        api_key: keys[provider],
        enabled: enabled[provider],
      })
      setMsgs((p) => ({ ...p, [provider]: 'บันทึกสำเร็จ' }))
      setKeys((p) => ({ ...p, [provider]: '' }))
      load()
    } catch (err) {
      setMsgs((p) => ({ ...p, [provider]: err.message }))
    }
  }

  return (
    <div className="bg-white rounded-lg shadow p-6 max-w-2xl mb-6">
      <h3 className="font-semibold text-gray-700 mb-1">OCR Engine Configuration</h3>
      <p className="text-xs text-gray-400 mb-5">ตั้งค่า API Key สำหรับ OCR dual-engine — บันทึกแล้วมีผลทันทีโดยไม่ต้อง restart</p>

      {configs.map((c) => (
        <div key={c.provider} className="border border-gray-200 rounded-lg p-4 mb-4">
          <div className="flex items-center justify-between mb-1">
            <div>
              <span className="font-medium text-gray-800 text-sm">{PROVIDER_LABEL[c.provider] ?? c.provider}</span>
              <p className="text-xs text-gray-400 mt-0.5">{PROVIDER_DESC[c.provider] ?? ''}</p>
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <span className="text-xs text-gray-500">{enabled[c.provider] ? 'เปิดใช้' : 'ปิด'}</span>
              <div
                onClick={() => setEnabled((p) => ({ ...p, [c.provider]: !p[c.provider] }))}
                className={`relative w-10 h-5 rounded-full transition-colors ${enabled[c.provider] ? 'bg-blue-600' : 'bg-gray-300'}`}
              >
                <div className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${enabled[c.provider] ? 'translate-x-5' : 'translate-x-0.5'}`} />
              </div>
            </label>
          </div>

          {c.api_key_masked && (
            <p className="text-xs text-gray-400 mb-2">
              Key ปัจจุบัน: <code className="bg-gray-100 px-1 rounded">{c.api_key_masked}</code>
            </p>
          )}

          <div className="flex gap-2 mt-2">
            <input
              type="password"
              placeholder="วาง API Key ใหม่ที่นี่ (เว้นว่างถ้าไม่ต้องการเปลี่ยน)"
              value={keys[c.provider] ?? ''}
              onChange={(e) => setKeys((p) => ({ ...p, [c.provider]: e.target.value }))}
              className="flex-1 border border-gray-300 rounded px-3 py-1.5 text-sm font-mono"
            />
            <Btn onClick={() => save(c.provider)}>บันทึก</Btn>
          </div>

          {msgs[c.provider] && (
            <p className={`text-xs mt-1.5 ${msgs[c.provider] === 'บันทึกสำเร็จ' ? 'text-green-600' : 'text-red-500'}`}>
              {msgs[c.provider]}
            </p>
          )}
        </div>
      ))}
    </div>
  )
}

// ─── OCR Test Panel ──────────────────────────────────────────────────────────

function fmt(val) {
  if (val == null) return '—'
  if (typeof val === 'number') return val.toLocaleString('th-TH', { minimumFractionDigits: 2 })
  return val || '—'
}

function DataRow({ label, gptVal, visionVal }) {
  const match = String(gptVal ?? '') === String(visionVal ?? '') || (!gptVal && !visionVal)
  return (
    <tr className="border-b border-gray-100">
      <td className="py-1.5 pr-3 text-xs text-gray-500 font-medium whitespace-nowrap">{label}</td>
      <td className="py-1.5 pr-3 text-xs text-gray-800 font-mono">{fmt(gptVal)}</td>
      <td className="py-1.5 text-xs text-gray-800 font-mono">
        <span className={!match && gptVal && visionVal ? 'text-red-500' : ''}>{fmt(visionVal)}</span>
      </td>
    </tr>
  )
}

function OCRTestSection() {
  const fileRef = useRef(null)
  const [file, setFile] = useState(null)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState(null)
  const [err, setErr] = useState('')
  const [rawExpanded, setRawExpanded] = useState(false)

  const run = async () => {
    if (!file) return
    setLoading(true)
    setResult(null)
    setErr('')
    try {
      const form = new FormData()
      form.append('file', file)
      const res = await api.post('/ocr/test', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      setResult(res.data.data)
    } catch (e) {
      setErr(e.message)
    } finally {
      setLoading(false)
    }
  }

  const gpt    = result?.gpt    ?? {}
  const vision = result?.vision ?? {}

  return (
    <div className="bg-white rounded-lg shadow p-6 max-w-3xl">
      <h3 className="font-semibold text-gray-700 mb-1">Test OCR Dual-Engine</h3>
      <p className="text-xs text-gray-400 mb-4">อัพโหลดภาพใบกำกับภาษี (JPG / PNG) เพื่อทดสอบผลจาก 2 engine พร้อมกัน</p>

      <div className="flex items-center gap-3 mb-4">
        <input ref={fileRef} type="file" accept=".jpg,.jpeg,.png" className="hidden"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
        <button
          onClick={() => fileRef.current?.click()}
          className="px-4 py-2 border-2 border-dashed border-gray-300 rounded text-sm text-gray-500 hover:border-blue-400 hover:text-blue-500 transition-colors"
        >
          {file ? file.name : 'เลือกไฟล์ภาพ'}
        </button>
        {file && <span className="text-xs text-gray-400">{(file.size / 1024).toFixed(0)} KB</span>}
        <Btn onClick={run} disabled={!file || loading}>
          {loading ? 'กำลังประมวลผล…' : 'รัน OCR'}
        </Btn>
        {file && <button onClick={() => { setFile(null); setResult(null); setErr('') }}
          className="text-xs text-gray-400 hover:text-red-500">ล้าง</button>}
      </div>

      {err && <p className="text-sm text-red-500 mb-4">{err}</p>}

      {result && (
        <>
          {/* Match badge */}
          <div className="flex items-center gap-2 mb-4">
            <span className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold ${result.matched ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
              {result.matched ? '✓ Cross-verify ผ่าน' : '✗ Cross-verify ไม่ผ่าน — รายการจะเข้า HITL'}
            </span>
            <span className="text-xs text-gray-400">engine: {result.engine}</span>
          </div>

          {/* Side-by-side comparison */}
          <div className="overflow-x-auto mb-4">
            <table className="w-full text-left">
              <thead>
                <tr className="border-b-2 border-gray-200">
                  <th className="pb-2 text-xs text-gray-400 font-medium pr-3 w-32">ฟิลด์</th>
                  <th className="pb-2 text-xs text-blue-600 font-semibold pr-3">GPT-4o-mini</th>
                  <th className="pb-2 text-xs text-purple-600 font-semibold">Google Vision</th>
                </tr>
              </thead>
              <tbody>
                <DataRow label="ชื่อผู้ขาย"      gptVal={gpt.vendor_name}      visionVal={vision.vendor_name} />
                <DataRow label="เลขผู้เสียภาษี"  gptVal={gpt.vendor_tax_id}    visionVal={vision.vendor_tax_id} />
                <DataRow label="ยอดก่อน VAT"     gptVal={gpt.total_before_vat} visionVal={vision.total_before_vat} />
                <DataRow label="VAT"              gptVal={gpt.vat_amount}       visionVal={vision.vat_amount} />
                <DataRow label="ยอดรวม"           gptVal={gpt.total_amount}     visionVal={vision.total_amount} />
              </tbody>
            </table>
          </div>

          {/* Line items from GPT */}
          {gpt.items?.length > 0 && (
            <div className="mb-4">
              <p className="text-xs font-semibold text-gray-600 mb-2">รายการสินค้า (จาก GPT-4o-mini)</p>
              <table className="w-full text-left">
                <thead>
                  <tr className="bg-gray-50 text-xs text-gray-500">
                    <th className="px-3 py-1.5 font-medium">รายการ</th>
                    <th className="px-3 py-1.5 font-medium text-right">จำนวน</th>
                    <th className="px-3 py-1.5 font-medium text-right">ราคา/หน่วย</th>
                    <th className="px-3 py-1.5 font-medium text-right">รวม</th>
                  </tr>
                </thead>
                <tbody>
                  {gpt.items.map((item, i) => (
                    <tr key={i} className="border-b border-gray-100 text-xs">
                      <td className="px-3 py-1.5 text-gray-800">{item.description || '—'}</td>
                      <td className="px-3 py-1.5 text-right text-gray-600">{item.quantity}</td>
                      <td className="px-3 py-1.5 text-right text-gray-600">{fmt(item.unit_price)}</td>
                      <td className="px-3 py-1.5 text-right text-gray-800 font-medium">{fmt(item.total_price)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Raw text toggle */}
          {result.raw_text && (
            <div>
              <button
                onClick={() => setRawExpanded((p) => !p)}
                className="text-xs text-blue-500 hover:underline mb-1"
              >
                {rawExpanded ? '▲ ซ่อน raw text' : '▼ ดู raw text จาก GCV'}
              </button>
              {rawExpanded && (
                <pre className="bg-gray-50 border border-gray-200 rounded p-3 text-xs text-gray-700 whitespace-pre-wrap max-h-64 overflow-y-auto font-mono">
                  {result.raw_text}
                </pre>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}

// ─── Page ────────────────────────────────────────────────────────────────────

export default function Settings() {
  return (
    <div>
      <PageHeader title="Settings" />
      <RewardSection />
      <OCREngineSection />
      <OCRTestSection />
    </div>
  )
}
