export function StatusBadge({ value }) {
  const colors = {
    active:     'bg-green-100 text-green-700',
    inactive:   'bg-gray-100 text-gray-500',
    pending:    'bg-yellow-100 text-yellow-700',
    processing: 'bg-blue-100 text-blue-700',
    done:       'bg-green-100 text-green-700',
    failed:     'bg-red-100 text-red-700',
    verified:   'bg-green-100 text-green-700',
    conflict:   'bg-red-100 text-red-700',
    resolved:   'bg-green-100 text-green-700',
    open:       'bg-blue-100 text-blue-700',
    closed:     'bg-gray-100 text-gray-500',
    asset:      'bg-purple-100 text-purple-700',
    expense:    'bg-orange-100 text-orange-700',
    paid:       'bg-green-100 text-green-700',
  }
  const cls = colors[value] ?? 'bg-gray-100 text-gray-600'
  return (
    <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${cls}`}>
      {value}
    </span>
  )
}

export function PageHeader({ title, action }) {
  return (
    <div className="flex items-center justify-between mb-5">
      <h2 className="text-xl font-semibold text-gray-800">{title}</h2>
      {action}
    </div>
  )
}

export function Btn({ children, onClick, variant = 'primary', type = 'button', disabled }) {
  const base = 'px-4 py-2 rounded text-sm font-medium transition-colors disabled:opacity-50'
  const variants = {
    primary:   'bg-blue-600 text-white hover:bg-blue-700',
    secondary: 'bg-gray-200 text-gray-700 hover:bg-gray-300',
    danger:    'bg-red-600 text-white hover:bg-red-700',
    success:   'bg-green-600 text-white hover:bg-green-700',
  }
  return (
    <button type={type} onClick={onClick} disabled={disabled} className={`${base} ${variants[variant]}`}>
      {children}
    </button>
  )
}

export function Input({ label, name, value, onChange, type = 'text', required }) {
  return (
    <div className="mb-4">
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {label}{required && <span className="text-red-500 ml-1">*</span>}
      </label>
      <input
        type={type}
        name={name}
        value={value}
        onChange={onChange}
        required={required}
        className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
    </div>
  )
}

export function Select({ label, name, value, onChange, options, required }) {
  return (
    <div className="mb-4">
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {label}{required && <span className="text-red-500 ml-1">*</span>}
      </label>
      <select
        name={name}
        value={value}
        onChange={onChange}
        required={required}
        className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        <option value="">— เลือก —</option>
        {options.map((o) => (
          <option key={o.value} value={o.value}>{o.label}</option>
        ))}
      </select>
    </div>
  )
}

export function useForm(initial) {
  const [form, setForm] = React.useState(initial)
  const handleChange = (e) => setForm((f) => ({ ...f, [e.target.name]: e.target.value }))
  const reset = () => setForm(initial)
  return [form, handleChange, reset, setForm]
}

import React from 'react'
