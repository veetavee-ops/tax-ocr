import { useEffect, useState } from 'react'
import api from '../api/client'
import { PageHeader, Btn, StatusBadge, useForm } from '../components/ui'

export default function StorageConfig() {
  const [tenants, setTenants] = useState([])
  const [config, setConfig]   = useState(null)
  const [tenantID, setTenantID] = useState('')
  const [form, onChange, , setForm] = useForm({
    storage_type: 'gdrive',
    gdrive_folder_id: '',
    gdrive_folder_url: '',
    onedrive_folder_id: '',
    onedrive_folder_url: '',
    owned_by: 'us',
    billing_type: 'included',
    monthly_fee: '0',
  })
  const [msg, setMsg] = useState('')

  useEffect(() => { api.get('/tenants').then((r) => setTenants(r.data.data ?? [])) }, [])

  const load = (id) => {
    setTenantID(id)
    api.get(`/storage/config/${id}`).then((r) => {
      const d = r.data.data
      if (d) { setConfig(d); setForm({ ...d, monthly_fee: String(d.monthly_fee) }) }
      else setConfig(null)
    }).catch(() => setConfig(null))
  }

  const submit = async (e) => {
    e.preventDefault(); setMsg('')
    try {
      const payload = { ...form, tenant_id: tenantID, monthly_fee: parseFloat(form.monthly_fee) }
      if (config) await api.put(`/storage/config/${tenantID}`, payload)
      else await api.post('/storage/config', payload)
      setMsg('บันทึกสำเร็จ'); load(tenantID)
    } catch (err) { setMsg(err.message) }
  }

  return (
    <div>
      <PageHeader title="Storage Config" />
      <div className="mb-5">
        <label className="block text-sm font-medium text-gray-700 mb-1">เลือก Tenant</label>
        <select onChange={(e) => load(e.target.value)} value={tenantID}
          className="border border-gray-300 rounded px-3 py-2 text-sm">
          <option value="">— เลือก Tenant —</option>
          {tenants.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
        </select>
      </div>

      {tenantID && (
        <form onSubmit={submit} className="bg-white rounded-lg shadow p-6 max-w-lg space-y-4">
          <h3 className="font-semibold text-gray-700">{config ? 'แก้ไข Config' : 'สร้าง Config ใหม่'}</h3>

          {[
            { label: 'Storage Type', name: 'storage_type', opts: ['gdrive','onedrive','both'] },
            { label: 'Owned By',     name: 'owned_by',     opts: ['us','tenant'] },
            { label: 'Billing Type', name: 'billing_type', opts: ['included','charged'] },
          ].map(({ label, name, opts }) => (
            <div key={name}>
              <label className="block text-sm font-medium text-gray-700 mb-1">{label}</label>
              <select name={name} value={form[name]} onChange={onChange}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm">
                {opts.map((o) => <option key={o} value={o}>{o}</option>)}
              </select>
            </div>
          ))}

          {['gdrive','both'].includes(form.storage_type) && <>
            {[['GDrive Folder ID','gdrive_folder_id'],['GDrive Folder URL','gdrive_folder_url']].map(([l,n]) => (
              <div key={n}><label className="block text-sm font-medium text-gray-700 mb-1">{l}</label>
                <input name={n} value={form[n]} onChange={onChange} className="w-full border border-gray-300 rounded px-3 py-2 text-sm" /></div>
            ))}
          </>}

          {['onedrive','both'].includes(form.storage_type) && <>
            {[['OneDrive Folder ID','onedrive_folder_id'],['OneDrive Folder URL','onedrive_folder_url']].map(([l,n]) => (
              <div key={n}><label className="block text-sm font-medium text-gray-700 mb-1">{l}</label>
                <input name={n} value={form[n]} onChange={onChange} className="w-full border border-gray-300 rounded px-3 py-2 text-sm" /></div>
            ))}
          </>}

          {form.billing_type === 'charged' && (
            <div><label className="block text-sm font-medium text-gray-700 mb-1">Monthly Fee (THB)</label>
              <input name="monthly_fee" value={form.monthly_fee} onChange={onChange} type="number"
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm" /></div>
          )}

          {msg && <p className={`text-sm ${msg === 'บันทึกสำเร็จ' ? 'text-green-600' : 'text-red-500'}`}>{msg}</p>}
          <Btn type="submit">บันทึก</Btn>
        </form>
      )}
    </div>
  )
}
