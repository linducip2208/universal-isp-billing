import React, { useEffect, useState } from 'react'
import { api } from '../api/client'
import { useLang } from '../main'

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
    return (
      <div className="page"><h2>{ctx.t('nav.noc')}</h2>
        {err
          ? <div className="empty">{ctx.t('noc.offline')}: {err}</div>
          : <div className="card skeleton">Loading…</div>}
      </div>
    )
  }
  const cards: [string, number][] = [
    ['Devices online', d.online_devices ?? 0], ['Devices offline', d.offline_devices ?? 0],
    ['Active subscribers', d.active_subscribers ?? 0], ['Suspended', d.suspended_subscribers ?? 0],
    ['Online sessions', d.online_sessions ?? 0], ['Critical alerts', d.critical_alerts ?? 0],
  ]
  return (
    <div className="page">
      <h2>{ctx.t('nav.noc')} — {live ? ctx.t('noc.live') : ctx.t('noc.offline')}</h2>
      <div className="cards">
        {cards.map(([k, v]) => <div className="card" key={k}><div className="ct">{k}</div><div className="cv">{v}</div></div>)}
      </div>
      <div className="card"><div className="ct">GET /api/v1/noc/summary</div><pre>{JSON.stringify(d, null, 2)}</pre></div>
    </div>
  )
}
