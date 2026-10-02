import React, { useEffect, useState } from 'react'
import { api } from '../api/client'
import { useLang } from '../main'

const seed = { total_devices: 12, online_devices: 10, offline_devices: 1, degraded_devices: 1, active_subscribers: 1240, suspended_subscribers: 37, online_sessions: 986, active_alerts: 4, critical_alerts: 1, provisioning_failures_24h: 2 }

export function NOC() {
  const ctx: any = useLang()
  const [d, setD] = useState<any>(seed)
  const [live, setLive] = useState(false)
  useEffect(() => {
    let stop = false
    async function poll() {
      try {
        const j = await api('/api/v1/noc/summary')
        if (!stop) { setD(j); setLive(true) }
      } catch { if (!stop) setLive(false) }
    }
    poll()
    const id = setInterval(poll, 5000)
    return () => { stop = true; clearInterval(id) }
  }, [])
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
