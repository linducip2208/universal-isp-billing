import React, { useEffect, useState } from 'react'
import { api } from '../api/client'
import { useLang } from '../lang'

function Table({ rows, cols }: { rows: any[]; cols: string[] }) {
  if (!rows.length) return <div className="empty">—</div>
  return (
    <table><thead><tr>{cols.map(c => <th key={c}>{c}</th>)}</tr></thead>
      <tbody>{rows.map((r, i) => <tr key={i}>{cols.map(c => <td key={c}>{String(r[c] ?? r[c.toLowerCase()] ?? '')}</td>)}</tr>)}</tbody></table>
  )
}

export function Invoices() {
  const [rows, setRows] = useState<any[]>([])
  useEffect(() => { api('/api/v1/invoices').then(j => setRows(j.data ?? [])).catch(() => {}) }, [])
  return (
    <div className="page"><h2>Invoices</h2>
      <div className="card"><Table rows={rows} cols={['id', 'status', 'total']} /></div>
    </div>
  )
}

export function Payments() {
  const [rows, setRows] = useState<any[]>([])
  useEffect(() => { api('/api/v1/payments').then(j => setRows(j.data ?? [])).catch(() => {}) }, [])
  return (
    <div className="page"><h2>Payments</h2>
      <div className="card"><Table rows={rows} cols={['id', 'status', 'amount']} /></div>
    </div>
  )
}

