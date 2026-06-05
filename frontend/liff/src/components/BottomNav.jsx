import React from 'react'
import { NavLink } from 'react-router-dom'

const tabs = [
  { to: '/upload',       icon: '📤', label: 'ส่งเอกสาร' },
  { to: '/status',       icon: '📋', label: 'สถานะ' },
  { to: '/conversation', icon: '💬', label: 'สนทนา' },
]

export default function BottomNav() {
  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 safe-bottom">
      <div className="flex">
        {tabs.map(t => (
          <NavLink
            key={t.to}
            to={t.to}
            className={({ isActive }) =>
              `flex-1 flex flex-col items-center py-2 text-xs gap-0.5 ${
                isActive ? 'text-line font-semibold' : 'text-gray-500'
              }`
            }
          >
            <span className="text-xl leading-none">{t.icon}</span>
            {t.label}
          </NavLink>
        ))}
      </div>
    </nav>
  )
}
