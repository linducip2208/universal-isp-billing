import React, { useState } from 'react'
import { api } from '../api/client'

function Sec({ title, rows, cols }: { title: string; rows: any[]; cols: string[] }) {
  if (!rows || !rows.length) return null
  return (
    <div className="card mb-3"><div className="card-header"><h3 className="card-title">{title} ({rows.length})</h3></div>
      <div className="table-responsive">
        <table className="table table-vcenter card-table table-striped">
          <thead><tr>{cols.map(c => <th key={c}>{c}</th>)}</tr></thead>
          <tbody>{rows.map((r: any, i: number) => (
            <tr key={i}>{cols.map(c => <td key={c}>{String(r[c] ?? '')}</td>)}</tr>
          ))}</tbody>
        </table>
      </div>
    </div>
  )
}

export function Customer360() {
  const [id, setId] = useState('')
  const [d, setD] = useState<any>(null)
  const [err, setErr] = useState('')
  async function go(e: React.FormEvent) {
    e.preventDefault(); setErr(''); setD(null)
    try { setD(await api(`/api/v1/customer-360?customer=${encodeURIComponent(id)}`)) }
    catch (e: any) { setErr(String(e.message || e)) }
  }
  return (
    <div>
      <h2 className="page-title mb-3">Customer 360</h2>
      {err && <div className="alert alert-danger" role="alert">{err}</div>}
      <form onSubmit={go} className="card mb-3">
        <div className="card-body">
          <div className="mb-3"><label className="form-label">Customer ID
            <input className="form-control" value={id} onChange={e => setId(e.target.value)} placeholder="uuid — pick from Customers list" />
          </label></div>
          <button className="btn btn-primary">Load</button>
        </div>
      </form>
      {d && <div className="card mb-3"><div className="card-header"><h3 className="card-title">Customer</h3></div>
        <div className="card-body"><pre className="debug">{JSON.stringify(d.customer, null, 2)}</pre></div></div>}
      {d && <>
        <Sec title="Subscriptions" rows={d.subscriptions} cols={['username', 'service', 'status']} />
        <Sec title="Invoices" rows={d.invoices} cols={['id', 'status']} />
        <Sec title="Payments" rows={d.payments} cols={['id', 'status']} />
        <Sec title="Tickets" rows={d.tickets} cols={['subject', 'status']} />
        <Sec title="Contacts" rows={d.contacts} cols={['kind', 'value']} />
        <Sec title="Addresses" rows={d.addresses} cols={['label', 'address']} />
      </>}
    </div>
  )
}

export function DeviceDetail() {
  const [id, setId] = useState('')
  const [d, setD] = useState<any>(null)
  const [runs, setRuns] = useState<any[]>([])
  const [err, setErr] = useState('')
  async function go(e: React.FormEvent) {
    e.preventDefault(); setErr(''); setD(null); setRuns([])
    try {
      const dev = await api(`/api/v1/devices?id=${encodeURIComponent(id)}`)
      setD(dev)
      const r = await api('/api/v1/lab/runs?per_page=5')
      setRuns((r.data ?? []).filter((x: any) => x.host === dev?.host))
    } catch (e: any) { setErr(String(e.message || e)) }
  }
  return (
    <div>
      <h2 className="page-title mb-3">Device Detail</h2>
      {err && <div className="alert alert-danger" role="alert">{err}</div>}
      <form onSubmit={go} className="card mb-3">
        <div className="card-body">
          <div className="mb-3"><label className="form-label">Device ID
            <input className="form-control" value={id} onChange={e => setId(e.target.value)} placeholder="uuid — pick from Devices list" />
          </label></div>
          <button className="btn btn-primary">Load</button>
        </div>
      </form>
      {d && <div className="card mb-3"><div className="card-header"><h3 className="card-title">Overview</h3></div>
        <div className="card-body"><pre className="debug">{JSON.stringify(d, null, 2)}</pre></div></div>}
      {runs.length > 0 && <div className="card"><div className="card-header"><h3 className="card-title">Recent lab evidence</h3></div>
        <div className="card-body"><pre className="debug">{JSON.stringify(runs, null, 2)}</pre></div></div>}
    </div>
  )
}

export function Incidents() {
  const [rows, setRows] = useState<any[]>([])
  const [err, setErr] = useState('')
  const [msg, setMsg] = useState('')
  async function load() {
    try { setRows((await api('/api/v1/incidents?per_page=25')).data ?? []) }
    catch (e: any) { setErr(String(e.message || e)) }
  }
  React.useEffect(() => { load() }, [])
  async function act(id: string, action: string) {
    const extra = action === 'assign' ? (prompt('Assignee:') || '') : action === 'resolve' ? (prompt('Root cause:') || '') : ''
    if ((action === 'assign' || action === 'resolve') && !extra) return
    if (!confirm(`${action} incident?`)) return
    try {
      await api('/api/v1/incidents/action', { method: 'POST', body: JSON.stringify({ id, action, extra }) })
      setMsg(`${action} ok`)
      load()
    } catch (e: any) { setErr(String(e.message || e)) }
  }
  return (
    <div>
      <h2 className="page-title mb-3">Incidents</h2>
      {err && <div className="alert alert-danger" role="alert">{err}</div>}
      {msg && <div className="alert alert-info" role="status">{msg}</div>}
      {rows.length === 0 && !err && (
        <div className="empty"><p className="empty-title">No incidents</p><p className="empty-subtitle">Empty state.</p></div>
      )}
      {rows.length > 0 && <div className="card">
        <div className="table-responsive">
          <table className="table table-vcenter card-table table-striped">
            <thead><tr><th>Title</th><th>Severity</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>{rows.map((r: any) => (
              <tr key={r.id}><td>{r.title}</td>
                <td><span className="badge bg-red-lt">{r.severity}</span></td><td>{r.status}</td>
                <td>
                  <button className="btn btn-sm me-1" onClick={() => act(r.id, 'ack')}>Ack</button>
                  <button className="btn btn-sm me-1" onClick={() => act(r.id, 'assign')}>Assign</button>
                  <button className="btn btn-sm" onClick={() => act(r.id, 'resolve')}>Resolve</button>
                </td></tr>
            ))}</tbody>
          </table>
        </div>
      </div>}
    </div>
  )
}
