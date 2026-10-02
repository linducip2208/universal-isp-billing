import React, { useEffect, useState } from 'react'
import { api } from '../api/client'
import { useLang } from '../lang'

export function Connectors() {
  const ctx: any = useLang()
  const [rows, setRows] = useState<any[]>([])
  useEffect(() => { api('/api/v1/connectors').then(j => setRows(j.data ?? [])).catch(() => {}) }, [])
  return (
    <div className="page">
      <h2>{ctx.t('conn.title')}</h2>
      <div className="card">
        <table><thead><tr><th>Vendor</th><th>Family</th><th>Conn</th><th>Status</th></tr></thead>
          <tbody>{rows.map((r, i) => (
            <tr key={i}><td>{r.Vendor}</td><td>{r.ProductFamily}</td><td>{r.ConnectionType}</td>
              <td><span className="badge ok">{r.Status}</span></td></tr>
          ))}</tbody></table>
      </div>
    </div>
  )
}

export function Matrix() {
  const ctx: any = useLang()
  const [rows, setRows] = useState<any[]>([])
  useEffect(() => { api('/api/v1/vendor-matrix').then(j => setRows(j.data ?? [])).catch(() => {}) }, [])
  return (
    <div className="page">
      <h2>{ctx.t('matrix.title')}</h2>
      <div className="card">
        <table><thead><tr><th>Vendor</th><th>Family</th><th>Conn</th><th>{ctx.t('matrix.status')}</th></tr></thead>
          <tbody>{rows.map((r, i) => (
            <tr key={i}><td>{r.Vendor}</td><td>{r.ProductFamily}</td><td>{r.ConnectionType}</td><td>{r.Status}</td></tr>
          ))}</tbody></table>
      </div>
    </div>
  )
}

