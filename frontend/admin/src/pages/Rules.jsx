import { useEffect, useState } from 'react'
import api from '../api/client'
import Table from '../components/Table'
import Modal from '../components/Modal'
import { PageHeader, Btn, Input, Select, StatusBadge, useForm } from '../components/ui'

const INIT = { tenant_id: '', keyword: '', asset_type: 'expense', source: 'human', confidence: '1.0' }

export default function Rules() {
  const [data, setData]         = useState([])
  const [tenants, setTenants]   = useState([])
  const [modal, setModal]       = useState(null)
  const [selected, setSelected] = useState(null)
  const [form, onChange, reset, setForm] = useForm(INIT)
  const [error, setError]       = useState('')
  const [testKw, setTestKw]     = useState('')
  const [testTenant, setTestTenant] = useState('')
  const [testResult, setTestResult] = useState(null)

  const load = () =>
    Promise.all([api.get('/rules'), api.get('/tenants')]).then(([r, t]) => {
      setData(r.data.data ?? [])
      setTenants(t.data.data ?? [])
    })
  useEffect(() => { load() }, [])

  const openCreate = () => { reset(); setError(''); setModal('create') }
  const openEdit   = (row) => {
    setForm({ tenant_id: row.tenant_id, keyword: row.keyword, asset_type: row.asset_type, source: row.source, confidence: String(row.confidence) })
    setSelected(row); setError(''); setModal('edit')
  }

  const submit = async (e) => {
    e.preventDefault(); setError('')
    try {
      const payload = { ...form, confidence: parseFloat(form.confidence) }
      if (modal === 'create') await api.post('/rules', payload)
      else await api.put(`/rules/${selected.id}`, { keyword: form.keyword, asset_type: form.asset_type, confidence: parseFloat(form.confidence) })
      setModal(null); load()
    } catch (err) { setError(err.message) }
  }

  const deleteRule = async (row) => {
    if (!confirm(`ลบ rule "${row.keyword}"?`)) return
    await api.delete(`/rules/${row.id}`)
    load()
  }

  const runTest = async () => {
    const r = await api.post('/rules/test', { tenant_id: testTenant, keyword: testKw })
    setTestResult(r.data)
  }

  const tenantOpts = tenants.map((t) => ({ value: t.id, label: t.name }))

  const cols = [
    { key: 'id',         label: 'ID',      render: (r) => <span className="font-mono text-xs text-gray-400">{r.id.slice(0,8)}…</span> },
    { key: 'keyword',    label: 'Keyword' },
    { key: 'asset_type', label: 'ประเภท',  render: (r) => <StatusBadge value={r.asset_type} /> },
    { key: 'source',     label: 'Source' },
    { key: 'confidence', label: 'Confidence', render: (r) => `${(r.confidence * 100).toFixed(0)}%` },
    { key: 'actions', label: '', render: (r) => (
      <button onClick={(e) => { e.stopPropagation(); deleteRule(r) }} className="text-xs text-red-500 hover:underline">ลบ</button>
    )},
  ]

  return (
    <div>
      <PageHeader title="Classification Rules" action={<Btn onClick={openCreate}>+ เพิ่ม Rule</Btn>} />

      {/* Test panel */}
      <div className="bg-white rounded-lg shadow p-4 mb-5">
        <p className="text-sm font-semibold text-gray-700 mb-2">ทดสอบ Rule</p>
        <div className="flex gap-2">
          <select value={testTenant} onChange={(e) => setTestTenant(e.target.value)}
            className="border border-gray-300 rounded px-3 py-2 text-sm">
            <option value="">— Tenant —</option>
            {tenants.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
          </select>
          <input value={testKw} onChange={(e) => setTestKw(e.target.value)} placeholder="คำที่ต้องการทดสอบ"
            className="flex-1 border border-gray-300 rounded px-3 py-2 text-sm" />
          <Btn onClick={runTest}>ทดสอบ</Btn>
        </div>
        {testResult && (
          <div className="mt-2 text-sm">
            {testResult.matched
              ? <span className="text-green-600">✓ Match: <strong>{testResult.rule?.keyword}</strong> → <StatusBadge value={testResult.rule?.asset_type} /></span>
              : <span className="text-gray-400">ไม่พบ rule ที่ตรงกัน</span>}
          </div>
        )}
      </div>

      <Table columns={cols} data={data} onRowClick={openEdit} />

      {modal && (
        <Modal title={modal === 'create' ? 'เพิ่ม Rule' : 'แก้ไข Rule'} onClose={() => setModal(null)}>
          <form onSubmit={submit}>
            {modal === 'create' && <Select label="Tenant" name="tenant_id" value={form.tenant_id} onChange={onChange} options={tenantOpts} required />}
            <Input label="Keyword" name="keyword" value={form.keyword} onChange={onChange} required />
            <Select label="ประเภท" name="asset_type" value={form.asset_type} onChange={onChange}
              options={[{ value: 'asset', label: 'Asset' }, { value: 'expense', label: 'Expense' }]} required />
            <Select label="Source" name="source" value={form.source} onChange={onChange}
              options={[{ value: 'human', label: 'Human' }, { value: 'ai', label: 'AI' }]} />
            <Input label="Confidence (0-1)" name="confidence" value={form.confidence} onChange={onChange} type="number" />
            {error && <p className="text-red-500 text-sm mb-3">{error}</p>}
            <div className="flex justify-end gap-2">
              <Btn variant="secondary" onClick={() => setModal(null)}>ยกเลิก</Btn>
              <Btn type="submit">บันทึก</Btn>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
