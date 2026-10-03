import React, { useState } from 'react'
import { api } from '../api/client'

export function Customers() {
  const [rows, setRows] = useState<any[]>([])
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [msg, setMsg] = useState('')
  const [q, setQ] = useState('')
  function load(search = '') {
    api(`/api/v1/customers?per_page=25&search=${encodeURIComponent(search)}`)
      .then(j => setRows(j.data ?? [])).catch((e: any) => setMsg(String(e.message || e)))
  }
  React.useEffect(() => { load() }, [])
  async function create(e: React.FormEvent) {
    e.preventDefault(); setMsg('')
    try {
      const j = await api('/api/v1/customers/create', {
        method: 'POST', body: JSON.stringify({ name, email, phone }),
      })
      setMsg(`created ${String(j.id).slice(0, 8)}`)
      setName(''); setEmail(''); setPhone('')
      load(q)
    } catch (e: any) { setMsg(String(e.message || e)) }
  }
  return (
    <div className="page"><h2>Customers</h2>
      {msg && <div className="card"><div className="ct">{msg}</div></div>}
      <form onSubmit={create} className="card form">
        <label>Name<input value={name} onChange={e => setName(e.target.value)} required /></label>
        <label>Email<input value={email} onChange={e => setEmail(e.target.value)} /></label>
        <label>Phone<input value={phone} onChange={e => setPhone(e.target.value)} /></label>
        <button>Add customer</button>
      </form>
      <div className="toolbar">
        <input placeholder="Search" value={q} onChange={e => setQ(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') load(q) }} />
        <button onClick={() => load(q)}>Search</button>
      </div>
      <div className="card">
        <table><thead><tr><th>Name</th><th>Email</th><th>Phone</th><th>Status</th></tr></thead>
          <tbody>{rows.map((r: any, i: number) => (
            <tr key={i}><td>{r.name}</td><td>{r.email}</td><td>{r.phone}</td>
              <td><span className="badge ok">{r.status}</span></td></tr>
          ))}</tbody></table>
        {rows.length === 0 && <div className="empty">No customers — empty state.</div>}
      </div>
    </div>
  )
}
