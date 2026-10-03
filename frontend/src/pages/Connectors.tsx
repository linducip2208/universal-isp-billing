import React, { useEffect, useState } from 'react'
import { api } from '../api/client'
import { useLang } from '../lang'

function JsonCard({ title, data }: { title: string; data: any }) {
  return (
    <div className="card"><div className="card-header"><h3 className="card-title">{title}</h3></div>
      <div className="card-body"><pre className="debug">{JSON.stringify(data, null, 2)}</pre></div></div>
  )
}

export function Connectors() {
  const ctx: any = useLang()
  const [rows, setRows] = useState<any[]>([])
  useEffect(() => { api('/api/v1/connectors').then(j => setRows(j.data ?? [])).catch(() => {}) }, [])
  return (
    <div>
      <h2 className="page-title mb-3">{ctx.t('conn.title')}</h2>
      <div className="card">
        <div className="table-responsive">
          <table className="table table-vcenter card-table table-striped">
            <thead><tr><th>Vendor</th><th>Family</th><th>Conn</th><th>Status</th></tr></thead>
            <tbody>{rows.map((r, i) => (
              <tr key={i}><td>{r.Vendor}</td><td>{r.ProductFamily}</td><td>{r.ConnectionType}</td>
                <td><span className="badge bg-blue-lt">{r.Status}</span></td></tr>
            ))}</tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

export function Matrix() {
  const ctx: any = useLang()
  const [rows, setRows] = useState<any[]>([])
  useEffect(() => { api('/api/v1/vendor-matrix').then(j => setRows(j.data ?? [])).catch(() => {}) }, [])
  return (
    <div>
      <h2 className="page-title mb-3">{ctx.t('matrix.title')}</h2>
      <div className="card">
        <div className="table-responsive">
          <table className="table table-vcenter card-table table-striped">
            <thead><tr><th>Vendor</th><th>Family</th><th>Conn</th><th>{ctx.t('matrix.status')}</th></tr></thead>
            <tbody>{rows.map((r, i) => (
              <tr key={i}><td>{r.Vendor}</td><td>{r.ProductFamily}</td><td>{r.ConnectionType}</td><td>{r.Status}</td></tr>
            ))}</tbody></table>
        </div>
      </div>
    </div>
  )
}

export { JsonCard }
