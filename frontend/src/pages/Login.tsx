import React, { useState } from 'react'
import { useLang } from '../lang'

export function Login() {
  const ctx: any = useLang()
  const [u, setU] = useState('admin')
  const [p, setP] = useState('secret')
  const [err, setErr] = useState('')
  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr('')
    try {
      const r = await fetch('/api/v1/auth/login', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: u, password: p }),
      })
      if (!r.ok) throw new Error('login failed')
      const j = await r.json()
      localStorage.setItem('isp_token', j.token)
      location.hash = '#/'
    } catch {
      setErr('login failed')
    }
  }
  return (
    <div className="page narrow">
      <h2>{ctx.t('auth.login')}</h2>
      <p className="muted">{ctx.t('auth.required')}</p>
      {err && <div className="empty">{err}</div>}
      <form onSubmit={submit} className="card form">
        <label>{ctx.t('auth.username')}<input value={u} onChange={e => setU(e.target.value)} /></label>
        <label>{ctx.t('auth.password')}<input type="password" value={p} onChange={e => setP(e.target.value)} /></label>
        <button type="submit">{ctx.t('auth.login')}</button>
      </form>
    </div>
  )
}

