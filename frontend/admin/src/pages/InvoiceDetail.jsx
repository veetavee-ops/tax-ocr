import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import api from '../api/client'
import Table from '../components/Table'
import { Btn, StatusBadge } from '../components/ui'

export default function InvoiceDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [invoice, setInvoice] = useState(null)
  const [items, setItems]     = useState([])

  useEffect(() => {
    api.get(`/invoices/${id}`).then((r) => setInvoice(r.data.data))
    api.get(`/invoices/${id}/items`).then((r) => setItems(r.data.data ?? []))
  }, [id])

  if (!invoice) return <p className="text-gray-400">กำลังโหลด…</p>

  const itemCols = [
    { key: 'description', label: 'รายการ' },
    { key: 'quantity',    label: 'จำนวน' },
    { key: 'unit_price',  label: 'ราคา/หน่วย', render: (r) => r.unit_price?.toLocaleString() },
    { key: 'total_price', label: 'รวม',         render: (r) => r.total_price?.toLocaleString() },
    { key: 'asset_type',  label: 'ประเภท',       render: (r) => <StatusBadge value={r.asset_type} /> },
    { key: 'classified_by', label: 'จัดโดย' },
  ]

  return (
    <div>
      <div className="flex items-center gap-3 mb-5">
        <Btn variant="secondary" onClick={() => navigate(-1)}>← กลับ</Btn>
        <h2 className="text-xl font-semibold text-gray-800">Invoice Detail</h2>
      </div>

      <div className="bg-white rounded-lg shadow p-5 mb-5 grid grid-cols-2 gap-4 text-sm">
        <div><span className="text-gray-500">ID:</span> <span className="font-mono">{invoice.id}</span></div>
        <div><span className="text-gray-500">Status:</span> <StatusBadge value={invoice.status} /></div>
        <div><span className="text-gray-500">Vendor Tax ID:</span> {invoice.vendor_tax_id || '—'}</div>
        <div><span className="text-gray-500">File Hash:</span> <span className="font-mono text-xs">{invoice.file_hash}</span></div>
        <div><span className="text-gray-500">ก่อน VAT:</span> {invoice.total_before_vat?.toLocaleString()}</div>
        <div><span className="text-gray-500">VAT:</span> {invoice.vat_amount?.toLocaleString()}</div>
        <div><span className="text-gray-500">รวม:</span> <strong>{invoice.total_amount?.toLocaleString()}</strong></div>
        <div><span className="text-gray-500">วันที่:</span> {new Date(invoice.created_at).toLocaleString('th-TH')}</div>
      </div>

      <h3 className="font-semibold text-gray-700 mb-3">Line Items ({items.length})</h3>
      <Table columns={itemCols} data={items} />
    </div>
  )
}
