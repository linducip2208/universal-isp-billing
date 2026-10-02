import React, { useState } from 'react'
import { api } from '../api/client'

export function Copilot() {
  const [q, setQ] = useState('')
  const [ans, setAns] = useState<any>(null)
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  async function ask(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true); setErr(''); setAns(null)
    try { setAns(await api('/api/v1/copilot/ask', { method: 'POST', body: JSON.stringify({ question: q }) })) }
    catch (e: any) { setErr(String(e.message || e)) }
    finally { setBusy(false) }
  }
  return (
    <div className="page"><h2>AI Copilot (read-only)</h2>
      <p className="muted">Answers cite evidence. It cannot execute actions — proposals only.</p>
      {err && <div className="empty">{err}</div>}
      <form onSubmit={ask} className="card form">
        <label>Question<input value={q} onChange={e => setQ(e.target.value)} placeholder="any subscriber sessions down?" /></label>
        <button disabled={busy}>{busy ? '…' : 'Ask'}</button>
      </form>
      {ans && <div className="card"><div className="ct">Answer</div><pre>{JSON.stringify(ans, null, 2)}</pre></div>}
    </div>
  )
}

export function ServiceHealth() {
  const [sub, setSub] = useState('')
  const [rep, setRep] = useState<any>(null)
  const [err, setErr] = useState('')
  async function go(e: React.FormEvent) {
    e.preventDefault(); setErr(''); setRep(null)
    try { setRep(await api(`/api/v1/service-health?subscriber=${encodeURIComponent(sub)}`)) }
    catch (e: any) { setErr(String(e.message || e)) }
  }
  return (
    <div className="page"><h2>Service Health</h2>
      {err && <div className="empty">{err}</div>}
      <form onSubmit={go} className="card form">
        <label>Subscription ID<input value={sub} onChange={e => setSub(e.target.value)} /></label>
        <button>Check</button>
      </form>
      {rep && <div className="card">
        <div className="ct">Overall: <span className="badge ok">{rep.overall}</span></div>
        <table><thead><tr><th>Source</th><th>Status</th><th>Detail</th></tr></thead>
          <tbody>{(rep.sections ?? []).map((s: any, i: number) => (
            <tr key={i}><td>{s.source}</td><td><span className="badge ok">{s.status}</span></td><td>{s.detail}</td></tr>
          ))}</tbody></table>
      </div>}
    </div>
  )
}

export function Economics() {
  const [d, setD] = useState<any>(null)
  const [err, setErr] = useState('')
  React.useEffect(() => {
    api('/api/v1/economics/summary').then(setD).catch((e: any) => setErr(String(e.message || e)))
  }, [])
  return (
    <div className="page"><h2>Network Economics</h2>
      {err && <div className="empty">{err}</div>}
      {!d && !err && <div className="card skeleton">Loading…</div>}
      {d && <div className="cards">
        <div className="card"><div className="ct">MRR (cents)</div><div className="cv">{d.mrr_cents}</div></div>
        <div className="card"><div className="ct">ARPU (cents)</div><div className="cv">{d.arpu_cents}</div></div>
        <div className="card"><div className="ct">Active subs</div><div className="cv">{d.active_subs}</div></div>
      </div>}
      {d && <div className="card"><pre>{JSON.stringify(d, null, 2)}</pre></div>}
    </div>
  )
}

