import { NavLink } from 'react-router-dom'

const links = [
  { to: '/dashboard',     label: 'Dashboard' },
  { to: '/tenants',       label: 'Tenants' },
  { to: '/branches',      label: 'Branches' },
  { to: '/users',         label: 'Users' },
  { to: '/invoices',      label: 'Invoices' },
  { to: '/vendors',       label: 'ทะเบียนผู้ขาย' },
  { to: '/hitl',          label: 'HITL Queue' },
  { to: '/rules',         label: 'Rules' },
  { to: '/conversations', label: 'Conversations' },
  { to: '/storage',       label: 'Storage Config' },
  { to: '/archive',       label: 'Archive' },
  { to: '/audit-logs',    label: 'Audit Logs' },
  { to: '/settings',      label: 'Settings' },
]

export default function Sidebar() {
  return (
    <aside className="w-56 bg-gray-900 text-white flex flex-col">
      <div className="px-4 py-5 text-xl font-bold tracking-wide border-b border-gray-700">
        TaxOCR
      </div>
      <nav className="flex-1 px-2 py-4 space-y-1 overflow-y-auto">
        {links.map((l) => (
          <NavLink
            key={l.to}
            to={l.to}
            className={({ isActive }) =>
              `block px-3 py-2 rounded text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-blue-600 text-white'
                  : 'text-gray-300 hover:bg-gray-700 hover:text-white'
              }`
            }
          >
            {l.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
