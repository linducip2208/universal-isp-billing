import React, { useContext, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { HashRouter, Routes, Route, Link, useLocation } from 'react-router-dom'
import type { Lang } from './i18n/dict'
import { dirOf } from './i18n/dict'
import { LangCtx, translate } from './lang'
import { Page } from './pages/Page'
import { Login } from './pages/Login'
import { NOC } from './pages/NOC'
import { Connectors, Matrix } from './pages/Connectors'
import { Lab } from './pages/Lab'
import { Invoices, Payments } from './pages/Billing'
import { Copilot, ServiceHealth, Economics } from './pages/Ops'
import { Customers } from './pages/Customers'
import { Customer360, DeviceDetail, Incidents } from './pages/Premium'
import { useLang } from './lang'
import tablerLtr from '@tabler/core/dist/css/tabler.min.css?url'
import tablerRtl from '@tabler/core/dist/css/tabler.rtl.min.css?url'
import './tabler-overrides.css'

const groups: { title: string; items: { to: string; key: string }[] }[] = [
  { title: 'DASHBOARD', items: [{ to: '/', key: 'nav.dashboard' }, { to: '/noc', key: 'nav.noc' }] },
  { title: 'CUSTOMERS', items: [{ to: '/customers', key: 'nav.customers' }, { to: '/subscriptions', key: 'Subscriptions' }, { to: '/packages', key: 'Packages' }] },
  { title: 'BILLING', items: [{ to: '/invoices', key: 'billing.invoices' }, { to: '/payments', key: 'billing.payments' }] },
  { title: 'NETWORK', items: [{ to: '/devices', key: 'Devices' }, { to: '/connectors', key: 'Connectors' }, { to: '/topology', key: 'Topology' }, { to: '/lab', key: 'Connection Lab' }] },
  { title: 'SERVICES', items: [{ to: '/radius', key: 'RADIUS' }, { to: '/sessions', key: 'Sessions' }, { to: '/vouchers', key: 'Vouchers' }] },
  { title: 'FTTH', items: [{ to: '/olt', key: 'OLT' }, { to: '/onu', key: 'ONU/ONT' }] },
  { title: 'MONITORING', items: [{ to: '/alerts', key: 'Alerts' }, { to: '/events', key: 'Events' }, { to: '/incidents', key: 'Incidents' }, { to: '/customer360', key: 'Customer 360' }, { to: '/device', key: 'Device Detail' }] },
  { title: 'OPERATIONS', items: [{ to: '/tickets', key: 'Tickets' }, { to: '/health', key: 'Service Health' }, { to: '/economics', key: 'Economics' }, { to: '/copilot', key: 'AI Copilot' }] },
  { title: 'AUTOMATION', items: [{ to: '/provisioning', key: 'Provisioning' }, { to: '/rules', key: 'Rules' }] },
  { title: 'SYSTEM', items: [{ to: '/matrix', key: 'Vendor Matrix' }, { to: '/settings', key: 'Settings' }] },
]

function Shell() {
  const [dark, setDark] = useState(false)
  const loc = useLocation()
  const ctx: any = useContext(LangCtx)
  const authed = !!localStorage.getItem('isp_token')
  function logout() { localStorage.removeItem('isp_token'); location.hash = '#/login' }
  React.useEffect(() => {
    document.documentElement.setAttribute('data-bs-theme', dark ? 'dark' : 'light')
  }, [dark])
  return (
    <div className="page">
      <aside className="navbar navbar-vertical navbar-expand-lg navbar-dark">
        <div className="container-fluid">
          <h1 className="navbar-brand navbar-brand-autodark">
            <span className="navbar-brand-text">Universal ISP</span>
          </h1>
          <div className="navbar-collapse">
            <ul className="navbar-nav">
              {groups.map(g => (
                <li className="nav-item" key={g.title}>
                  <div className="nav-link nav-group-title">{g.title}</div>
                  <ul className="navbar-nav">
                    {g.items.map(it => (
                      <li className="nav-item" key={it.to}>
                        <Link to={it.to} className={loc.pathname === it.to ? 'nav-link active' : 'nav-link'}>
                          <span className="nav-link-title">{ctx.t(it.key)}</span>
                        </Link>
                      </li>
                    ))}
                  </ul>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </aside>
      <div className="page-wrapper">
        <header className="page-header d-print-none">
          <div className="container-xl">
            <div className="row g-2 align-items-center">
              <div className="col">
                <h2 className="page-title">{ctx.t('noc.title')}</h2>
              </div>
              <div className="col-auto ms-auto d-print-none">
                <div className="btn-list">
                  {!authed
                    ? <Link to="/login" className="btn">{ctx.t('auth.login')}</Link>
                    : <button className="btn" onClick={logout}>{ctx.t('auth.logout')}</button>}
                  <button className="btn btn-icon" onClick={() => ctx.setLang(ctx.lang === 'en' ? 'id' : ctx.lang === 'id' ? 'ar' : 'en')} title="language">{ctx.lang.toUpperCase()}</button>
                  <button className="btn btn-icon" onClick={() => setDark(!dark)} title="theme">{dark ? '☀' : '☾'}</button>
                </div>
              </div>
            </div>
          </div>
        </header>
        <div className="page-body">
          <div className="container-xl">
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/noc" element={<NOC />} />
              <Route path="/connectors" element={<Connectors />} />
              <Route path="/matrix" element={<Matrix />} />
              <Route path="/lab" element={<Lab />} />
              <Route path="/invoices" element={<Invoices />} />
              <Route path="/payments" element={<Payments />} />
              <Route path="/copilot" element={<Copilot />} />
              <Route path="/health" element={<ServiceHealth />} />
              <Route path="/economics" element={<Economics />} />
              <Route path="/customer360" element={<Customer360 />} />
              <Route path="/device" element={<DeviceDetail />} />
              <Route path="/incidents" element={<Incidents />} />
              <Route path="/customers" element={<Customers />} />
              <Route path="/" element={<Page k="nav.dashboard" />} />
              <Route path="*" element={<Page k={loc.pathname} />} />
            </Routes>
          </div>
        </div>
      </div>
    </div>
  )
}

function useTablerCSS(rtl: boolean) {
  React.useEffect(() => {
    const id = 'tabler-css'
    document.getElementById(id)?.remove()
    const link = document.createElement('link')
    link.id = id
    link.rel = 'stylesheet'
    link.href = rtl ? tablerRtl : tablerLtr
    document.head.appendChild(link)
    return () => { document.getElementById(id)?.remove() }
  }, [rtl])
}

function App() {
  const [lang, setLang] = useState<Lang>('en')
  const t = (k: string) => translate(lang, k)
  const rtl = dirOf(lang) === 'rtl'
  useTablerCSS(rtl)
  React.useEffect(() => {
    document.documentElement.dir = dirOf(lang)
    document.documentElement.lang = lang
  }, [lang])
  return (
    <LangCtx.Provider value={{ lang, setLang, t }}>
      <HashRouter><Shell /></HashRouter>
    </LangCtx.Provider>
  )
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>)
