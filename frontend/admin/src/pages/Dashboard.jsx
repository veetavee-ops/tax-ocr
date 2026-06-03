import { useEffect, useState } from 'react'
import api from '../api/client'
import { StatusBadge } from '../components/ui'

function StatCard({ label, value, color = 'blue' }) {
  const colors = {
    blue:   'border-l-blue-500',
    green:  'border-l-green-500',
    yellow: 'border-l-yellow-500',
    red:    'border-l-red-500',
  }
  return (
    <div className={`bg-white rounded-lg shadow p-5 border-l-4 ${colors[color]}`}>
      <p className="text-sm text-gray-500">{label}</p>
      <p className="text-3xl font-bold text-gray-800 mt-1">{value}</p>
    </div>
  )
}

export default function Dashboard() {
  const [stats, setStats] = useState({ tenants: 0, invoices: 0, hitl: 0, reviewers: 0 })
  const [recentInvoices, setRecentInvoices] = useState([])

  useEffect(() => {
    Promise.allSettled([
      api.get('/tenants'),
      api.get('/invoices'),
      api.get('/hitl/queue'),
      api.get('/reviewers'),
    ]).then(([t, inv, hitl, rev]) => {
      setStats({
        tenants:   t.value?.data?.data?.length ?? 0,
        invoices:  inv.value?.data?.data?.length ?? 0,
        hitl:      hitl.value?.data?.data?.length ?? 0,
        reviewers: rev.value?.data?.data?.length ?? 0,
      })
      setRecentInvoices((inv.value?.data?.data ?? []).slice(0, 5))
    })
  }, [])

  return (
    <div>
      <h2 className="text-xl font-semibold text-gray-800 mb-5">Dashboard</h2>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatCard label="Tenants"       value={stats.tenants}   color="blue" />
        <StatCard label="Invoices"      value={stats.invoices}  color="green" />
        <StatCard label="HITL Queue"    value={stats.hitl}      color="yellow" />
        <StatCard label="Reviewers"     value={stats.reviewers} color="red" />
      </div>

      <div className="bg-white rounded-lg shadow p-5">
        <h3 className="font-semibold text-gray-700 mb-3">Recent Invoices</h3>
        {recentInvoices.length === 0 ? (
          <p className="text-sm text-gray-400">ยังไม่มีข้อมูล</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 border-b">
                <th className="pb-2">ID</th>
                <th className="pb-2">Vendor Tax ID</th>
                <th className="pb-2">Total</th>
                <th className="pb-2">Status</th>
              </tr>
            </thead>
            <tbody>
              {recentInvoices.map((inv) => (
                <tr key={inv.id} className="border-b last:border-0">
                  <td className="py-2 font-mono text-xs text-gray-500">{inv.id?.slice(0, 8)}…</td>
                  <td className="py-2">{inv.vendor_tax_id || '—'}</td>
                  <td className="py-2">{inv.total_amount?.toLocaleString()}</td>
                  <td className="py-2"><StatusBadge value={inv.status} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
