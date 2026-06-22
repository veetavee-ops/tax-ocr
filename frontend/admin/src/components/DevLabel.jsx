import { useLocation } from 'react-router-dom'

const PAGES = [
  { match: (p) => p === '/dashboard',           label: 'P-00 Dashboard' },
  { match: (p) => p === '/tenants',             label: 'P-01 Tenants' },
  { match: (p) => p === '/branches',            label: 'P-02 Branches' },
  { match: (p) => p === '/users',               label: 'P-03 Users' },
  { match: (p) => p === '/invoices',            label: 'P-04 Invoices' },
  { match: (p) => p.startsWith('/invoices/'),   label: 'P-05 InvoiceDetail' },
  { match: (p) => p === '/vendors',             label: 'P-06 Vendors' },
  { match: (p) => p === '/hitl',               label: 'P-07 HitlQueue' },
  { match: (p) => p === '/rules',              label: 'P-08 Rules' },
  { match: (p) => p === '/conversations',      label: 'P-09 Conversations' },
  { match: (p) => p === '/storage',            label: 'P-10 StorageConfig' },
  { match: (p) => p === '/archive',            label: 'P-11 Archive' },
  { match: (p) => p === '/audit-logs',         label: 'P-12 AuditLogs' },
  { match: (p) => p === '/settings',           label: 'P-13 Settings' },
]

export default function DevLabel() {
  const { pathname } = useLocation()
  const page = PAGES.find((p) => p.match(pathname))
  return (
    <div className="fixed bottom-3 left-3 z-[9999] select-none pointer-events-none">
      <span className="bg-black/70 text-white text-[10px] font-mono px-2 py-0.5 rounded">
        {page ? page.label : pathname}
      </span>
    </div>
  )
}
