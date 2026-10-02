import React, { useState } from 'react'
import { api } from '../api/client'
import { useLang } from '../lang'

export function Lab() {
  const ctx: any = useLang()
  const [vendor, setVendor] = useState('MikroTik')
  const [family, setFamily] = useState('RouterOS')
  const [conn, setConn] = useState('cli')
  const [host, setHost] = useState('')
  const [user, setUser] = useState('admin')
  const [pass, setPass] = useState('')
  const [out, setOut] = useState<any>(null)
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true); setErr(''); setOut(null)
    try {
      const j = await api('/api/v1/lab/test', {
        method: 'POST',
        body: JSON.stringify({ vendor, family, connection_type: conn, config: { host, username: user, password: pass } }),
      })
      setOut(j)
    } catch (e: any) { setErr(e.message) }
    finally { setBusy(false) }
  }
  return (
    <div className="page">
      <h2>{ctx.t('lab.title')}</h2>
      <p className="muted">{ctx.t('lab.hint')}</p>
      {err && <div className="empty">{err}</div>}
      <form onSubmit={submit} className="card form">
        <label>{ctx.t('lab.vendor')}<input value={vendor} onChange={e => setVendor(e.target.value)} /></label>
        <label>{ctx.t('lab.family')}<input value={family} onChange={e => setFamily(e.target.value)} /></label>
        <label>{ctx.t('lab.conn')}
          <select value={conn} onChange={e => setConn(e.target.value)}>
            {['cli', 'rest', 'cloud_api', 'restconf', 'netconf', 'ssh', 'radius', 'snmp', 'webhook', 'generic_http'].map(c => <option key={c} value={c}>{c}</option>)}
          </select></label>
        <label>{ctx.t('lab.host')}<input value={host} onChange={e => setHost(e.target.value)} placeholder="192.168.88.1" /></label>
        <label>{ctx.t('lab.user')}<input value={user} onChange={e => setUser(e.target.value)} /></label>
        <label>{ctx.t('lab.pass')}<input type="password" value={pass} onChange={e => setPass(e.target.value)} /></label>
        <button type="submit" disabled={busy}>{busy ? '…' : ctx.t('common.test')}</button>
      </form>
      {out && <div className="card"><div className="ct">{ctx.t('lab.result')}</div><pre>{JSON.stringify(out, null, 2)}</pre></div>}
    </div>
  )
}

