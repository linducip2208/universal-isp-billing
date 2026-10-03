import React, { useEffect, useState } from 'react'
import { api } from '../api/client'
import { useLang } from '../lang'

export function NOC() {
  const ctx: any = useLang()
  const [d, setD] = useState<any>(null)
  const [live, setLive] = useState(false)
  const [err, setErr] = useState('')
  useEffect(() => {
    let stop = false
    async function poll() {
      try {
        const j = await api('/api/v1/noc/summary')
        if (!stop) { setD(j); setLive(true); setErr('') }
      } catch (e: any) { if (!stop) { setLive(false); setErr(String(e.message || e)) } }
    }
    poll()
    const id = setInterval(poll, 5000)
    return () => { stop = true; clearInterval(id) }
  }, [])
  if (!d) {
    return err
      ? <div className="alert alert-warning" role="alert">{ctx.t('noc.offline')}: {err}</div>
      : <div className="card"><div className="card-body"><div className="placeholder-glow"><span className="placeholder col-12" /></div></div></div>
  }
  const cards: [string, number][] = [
    ['Devices online', d.online_devices ?? 0], ['Devices offline', d.offline_devices ?? 0],
    ['Active subscribers', d.active_subscribers ?? 0], ['Suspended', d.suspended_subscribers ?? 0],
    ['Online sessions', d.online_sessions ?? 0], ['Critical alerts', d.critical_alerts ?? 0],
  ]
  const dist: [string, number, string][] = [
    ['online', d.online_devices ?? 0, '#2f9e44'],
    ['offline', d.offline_devices ?? 0, '#e03131'],
    ['degraded', d.degraded_devices ?? 0, '#f08c00'],
  ]
  const max = Math.max(1, ...dist.map(x => x[1] as number))
  return (
    <div>
      <div className="mb-2 text-secondary">{ctx.t('nav.noc')} — {live ? ctx.t('noc.live') : ctx.t('noc.offline')}</div>
      <div className="row row-deck row-cards mb-3">
        {cards.map(([k, v]) => (
          <div className="col-sm-6 col-lg-2" key={k}>
            <div className="card"><div className="card-body">
              <div className="subheader">{k}</div><div className="h1 mb-0">{v}</div>
            </div></div>
          </div>
        ))}
      </div>
      <div className="card mb-3">
        <div className="card-header"><h3 className="card-title">Device status (live)</h3></div>
        <div className="card-body">
          <svg width="100%" height={dist.length * 30 + 10} role="img">
            {dist.map(([k, v, c], i) => (
              <g key={k}>
                <text x="0" y={i * 30 + 20} fontSize="12" fill="currentColor">{k}</text>
                <rect x="90" y={i * 30 + 6} width={`${(Number(v) / max) * 60}%`} height="16" fill={String(c)} rx="4" />
                <text x="92%" y={i * 30 + 20} fontSize="12" fill="currentColor" textAnchor="end">{v}</text>
              </g>
            ))}
          </svg>
        </div>
      </div>
      <div className="card"><div className="card-header"><h3 className="card-title">GET /api/v1/noc/summary</h3></div>
        <div className="card-body"><pre className="debug">{JSON.stringify(d, null, 2)}</pre></div></div>
    </div>
  )
}
