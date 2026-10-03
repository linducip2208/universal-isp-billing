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
  const [msg, setMsg] = useState('')
  function load() {
    api('/api/v1/invoices').then(j => setRows(j.data ?? [])).catch(() => {})
  }
  useEffect(() => { load() }, [])
  async function pay(inv: any) {
    const amount = Number(inv.total_cents ?? inv.total ?? 0)
    if (!amount || !confirm(`Record manual payment of ${amount} cents for ${inv.id}?`)) return
    try {
      await api('/api/v1/payments/create', {
        method: 'POST',
        body: JSON.stringify({ invoice_id: inv.id, amount_cents: amount, method: 'manual', provider: 'manual', reference: `MAN-${Date.now()}`, idempotency_key: `ui-${inv.id}` }),
      })
      setMsg('payment recorded')
      load()
    } catch (e: any) { setMsg(String(e.message || e)) }
  }
  return (
    <div className="page"><h2>Invoices</h2>
      {msg && <div className="card"><div className="ct">{msg}</div></div>}
      <div className="card">
        <table><thead><tr><th>ID</th><th>Status</th><th>Total</th><th>Action</th></tr></thead>
          <tbody>{rows.map((r: any, i: number) => (
            <tr key={i}><td>{String(r.id).slice(0, 8)}</td>
              <td><span className="badge ok">{r.status}</span></td>
              <td>{String(r.total_cents ?? r.total ?? '')}</td>
              <td>{r.status !== 'paid' && <button onClick={() => pay(r)}>Pay</button>}</td></tr>
          ))}</tbody></table>
        {rows.length === 0 && <div className="empty">—</div>}
      </div>
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

