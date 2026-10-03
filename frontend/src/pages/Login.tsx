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
    <div className="row justify-content-center">
      <div className="col-md-6 col-lg-4">
        <div className="card card-md">
          <div className="card-body">
            <h2 className="card-title text-center mb-4">{ctx.t('auth.login')}</h2>
            <p className="text-secondary text-center">{ctx.t('auth.required')}</p>
            {err && <div className="alert alert-danger" role="alert">{err}</div>}
            <form onSubmit={submit}>
              <div className="mb-3">
                <label className="form-label">{ctx.t('auth.username')}
                  <input className="form-control" value={u} onChange={e => setU(e.target.value)} />
                </label>
              </div>
              <div className="mb-3">
                <label className="form-label">{ctx.t('auth.password')}
                  <input className="form-control" type="password" value={p} onChange={e => setP(e.target.value)} />
                </label>
              </div>
              <button className="btn btn-primary w-100" type="submit">{ctx.t('auth.login')}</button>
            </form>
          </div>
        </div>
      </div>
    </div>
  )
}
