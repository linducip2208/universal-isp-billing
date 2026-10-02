import React, { useEffect, useState } from 'react'

export function Page({ k }: { k: string }) {
  const [data, setData] = useState<any>(null)
  const [err, setErr] = useState('')
  useEffect(() => {
    fetch('/api/v1/noc/summary', { headers: { Authorization: 'Bearer demo' } })
      .then(r => (r.ok ? r.json() : Promise.reject(r.statusText)))
      .then(setData)
      .catch(() => setErr('API offline — showing seeded demo state'))
  }, [k])
  return (
    <div className="page">
      <h2>{k}</h2>
      {err && <div className="empty">{err}</div>}
      <div className="cards">
        {['Active subscribers', 'Online sessions', 'Online devices', 'Critical alerts'].map((c, i) => (
          <div className="card" key={c}><div className="ct">{c}</div><div className="cv">{[1240, 986, 10, 1][i]}</div></div>
        ))}
      </div>
      <div className="card">
        <div className="ct">Live summary (GET /api/v1/noc/summary)</div>
        <pre>{JSON.stringify(data ?? { seeded: true }, null, 2)}</pre>
      </div>
      <div className="card">
        <div className="ct">Table</div>
        <table><thead><tr><th>Name</th><th>Status</th><th>Actions</th></tr></thead>
          <tbody><tr><td>Demo row</td><td><span className="badge ok">active</span></td><td>View</td></tr></tbody></table>
      </div>
    </div>
  )
}
