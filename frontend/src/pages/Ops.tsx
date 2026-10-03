import React, { useState } from 'react'
import { api } from '../api/client'

function FormCard({ title, hint, children, onSubmit, busy, submit }: {
  title: string; hint?: string; children: React.ReactNode;
  onSubmit: (e: React.FormEvent) => void; busy?: boolean; submit: string;
}) {
  return (
    <div>
      <h2 className="page-title mb-3">{title}</h2>
      {hint && <p className="text-secondary">{hint}</p>}
      <form onSubmit={onSubmit} className="card">
        <div className="card-body">{children}
          <button className="btn btn-primary" type="submit" disabled={busy}>{busy ? '…' : submit}</button>
        </div>
      </form>
    </div>
  )
}

const F = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="mb-3"><label className="form-label">{label}{children}</label></div>
)
const I = (p: React.InputHTMLAttributes<HTMLInputElement>) => <input className="form-control" {...p} />

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
    <div>
      <FormCard title="AI Copilot (read-only)" hint="Answers cite evidence. It cannot execute actions — proposals only."
        onSubmit={ask} busy={busy} submit="Ask">
        <F label="Question"><I value={q} onChange={e => setQ(e.target.value)} placeholder="any subscriber sessions down?" /></F>
      </FormCard>
      {err && <div className="alert alert-danger mt-3" role="alert">{err}</div>}
      {ans && <div className="card mt-3"><div className="card-header"><h3 className="card-title">Answer</h3></div>
        <div className="card-body"><pre className="debug">{JSON.stringify(ans, null, 2)}</pre></div></div>}
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
    <div>
      <FormCard title="Service Health" onSubmit={go} submit="Check">
        <F label="Subscription ID"><I value={sub} onChange={e => setSub(e.target.value)} /></F>
      </FormCard>
      {err && <div className="alert alert-danger mt-3" role="alert">{err}</div>}
      {rep && <div className="card mt-3">
        <div className="card-header"><h3 className="card-title">Overall: <span className="badge bg-green-lt">{rep.overall}</span></h3></div>
        <div className="table-responsive">
          <table className="table table-vcenter card-table">
            <thead><tr><th>Source</th><th>Status</th><th>Detail</th></tr></thead>
            <tbody>{(rep.sections ?? []).map((s: any, i: number) => (
              <tr key={i}><td>{s.source}</td><td><span className="badge bg-green-lt">{s.status}</span></td><td>{s.detail}</td></tr>
            ))}</tbody>
          </table>
        </div>
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
  if (err) return <div className="alert alert-danger" role="alert">{err}</div>
  if (!d) return <div className="card"><div className="card-body"><div className="placeholder-glow"><span className="placeholder col-12" /></div></div></div>
  return (
    <div>
      <h2 className="page-title mb-3">Network Economics</h2>
      <div className="row row-deck row-cards mb-3">
        {[['MRR (cents)', d.mrr_cents], ['ARPU (cents)', d.arpu_cents], ['Active subs', d.active_subs]].map(([k, v]) => (
          <div className="col-sm-6 col-lg-4" key={String(k)}>
            <div className="card"><div className="card-body">
              <div className="subheader">{k}</div><div className="h1 mb-0">{v}</div>
            </div></div>
          </div>
        ))}
      </div>
      <div className="card"><div className="card-body"><pre className="debug">{JSON.stringify(d, null, 2)}</pre></div></div>
    </div>
  )
}
