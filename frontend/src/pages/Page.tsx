import React, { useEffect, useState } from 'react'
import { api } from '../api/client'

// Route -> API resource mapping. Unknown routes render an honest
// "not wired yet" state instead of fabricated rows.
const ROUTES: Record<string, { api: string; cols: string[] }> = {
  '/customers': { api: '/api/v1/customers', cols: ['name', 'email', 'phone', 'status'] },
  '/subscriptions': { api: '/api/v1/subscriptions', cols: ['username', 'service', 'status'] },
  '/packages': { api: '/api/v1/packages', cols: ['name', 'price_cents', 'service'] },
  '/devices': { api: '/api/v1/devices', cols: ['vendor', 'model', 'host', 'status'] },
  '/alerts': { api: '/api/v1/alerts', cols: ['severity', 'title', 'status'] },
  '/events': { api: '/api/v1/events', cols: ['type', 'actor', 'resource'] },
  '/provisioning': { api: '/api/v1/provisioning/jobs', cols: ['kind', 'status', 'attempts'] },
  '/topology': { api: '/api/v1/topology', cols: [] },
  '/incidents': { api: '/api/v1/incidents', cols: ['title', 'severity', 'status'] },
  '/tickets': { api: '/api/v1/tickets', cols: ['subject', 'status', 'priority'] },
  '/sessions': { api: '/api/v1/radius/sessions', cols: ['username', 'nas_ip', 'framed_ip'] },
  '/contracts': { api: '/api/v1/contracts', cols: ['kind', 'status', 'mrc_cents'] },
}

export function Page({ k }: { k: string }) {
  const [rows, setRows] = useState<any[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const route = ROUTES[k]

  useEffect(() => {
    if (!route) { setLoading(false); return }
    setLoading(true); setErr('')
    api(`${route.api}?page=${page}&per_page=15&search=${encodeURIComponent(q)}`)
      .then(j => {
        if (j.nodes) { setRows(j.nodes); setTotal(j.nodes.length) }
        else { setRows(j.data ?? []); setTotal(j.total ?? 0) }
      })
      .catch((e: any) => setErr(String(e.message || e)))
      .finally(() => setLoading(false))
  }, [k, page, q])

  if (!route) {
    return (
      <div className="empty">
        <p className="empty-title">{k}</p>
        <p className="empty-subtitle">Module API not wired yet — tracked as PLANNED, no fake data shown.</p>
      </div>
    )
  }
  return (
    <div>
      <div className="card mb-3">
        <div className="card-body">
          <div className="d-flex gap-2">
            <input className="form-control" placeholder="Search" value={search}
              onChange={e => setSearch(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') { setPage(1); setQ(search) } }} />
            <button className="btn" onClick={() => { setPage(1); setQ(search) }}>Search</button>
          </div>
        </div>
      </div>
      {loading && <div className="card"><div className="card-body"><div className="placeholder-glow"><span className="placeholder col-12" /></div></div></div>}
      {!loading && err && <div className="alert alert-warning" role="alert">API error: {err} (empty state — nothing fabricated)</div>}
      {!loading && !err && rows.length === 0 && (
        <div className="empty"><p className="empty-title">No data</p><p className="empty-subtitle">Empty state.</p></div>
      )}
      {!loading && !err && rows.length > 0 && (
        <div className="card">
          <div className="table-responsive">
            <table className="table table-vcenter card-table table-striped">
              <thead><tr>{route.cols.map(c => <th key={c}>{c}</th>)}</tr></thead>
              <tbody>{rows.map((r: any, i: number) => (
                <tr key={i}>{route.cols.map(c => (
                  <td key={c}>{c === 'status' || c === 'severity'
                    ? <span className="badge bg-green-lt">{String(r[c] ?? '')}</span>
                    : String(r[c] ?? '')}</td>
                ))}</tr>
              ))}</tbody>
            </table>
          </div>
          <div className="card-footer d-flex align-items-center gap-2">
            <button className="btn btn-sm" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>‹</button>
            <span>{page} / {Math.max(1, Math.ceil(total / 15))} ({total})</span>
            <button className="btn btn-sm" disabled={page * 15 >= total} onClick={() => setPage(p => p + 1)}>›</button>
          </div>
        </div>
      )}
    </div>
  )
}
