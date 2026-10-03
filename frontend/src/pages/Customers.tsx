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
    <div>
      <h2 className="page-title mb-3">Customers</h2>
      {msg && <div className="alert alert-info" role="status">{msg}</div>}
      <form onSubmit={create} className="card mb-3">
        <div className="card-body">
          <div className="row g-2">
            <div className="col-md-4"><label className="form-label">Name<input className="form-control" value={name} onChange={e => setName(e.target.value)} required /></label></div>
            <div className="col-md-4"><label className="form-label">Email<input className="form-control" value={email} onChange={e => setEmail(e.target.value)} /></label></div>
            <div className="col-md-3"><label className="form-label">Phone<input className="form-control" value={phone} onChange={e => setPhone(e.target.value)} /></label></div>
            <div className="col-md-1 d-flex align-items-end"><button className="btn btn-primary">Add customer</button></div>
          </div>
        </div>
      </form>
      <div className="card mb-3"><div className="card-body">
        <div className="d-flex gap-2">
          <input className="form-control" placeholder="Search" value={q} onChange={e => setQ(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') load(q) }} />
          <button className="btn" onClick={() => load(q)}>Search</button>
        </div>
      </div></div>
      <div className="card">
        <div className="table-responsive">
          <table className="table table-vcenter card-table table-striped">
            <thead><tr><th>Name</th><th>Email</th><th>Phone</th><th>Status</th></tr></thead>
            <tbody>{rows.map((r: any, i: number) => (
              <tr key={i}><td>{r.name}</td><td>{r.email}</td><td>{r.phone}</td>
                <td><span className="badge bg-green-lt">{r.status}</span></td></tr>
            ))}</tbody>
          </table>
        </div>
        {rows.length === 0 && <div className="card-body"><div className="empty"><p className="empty-title">No customers</p><p className="empty-subtitle">Empty state.</p></div></div>}
      </div>
    </div>
  )
}
