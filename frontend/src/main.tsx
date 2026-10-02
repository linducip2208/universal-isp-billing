import React, { createContext, useContext, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { HashRouter, Routes, Route, Link, useLocation } from 'react-router-dom'
import { en, id, Lang } from './i18n/dict'
import { Page } from './pages/Page'
import { Login } from './pages/Login'
import { NOC } from './pages/NOC'
import { Connectors, Matrix } from './pages/Connectors'
import { Lab } from './pages/Lab'
import { Invoices, Payments } from './pages/Billing'
import { Copilot, ServiceHealth, Economics } from './pages/Ops'

const LangCtx = createContext<{ lang: Lang; setLang: (l: Lang) => void; t: (k: string) => void } | any>({})
export const useLang = () => useContext(LangCtx)

const groups: { title: string; items: { to: string; key: string }[] }[] = [
  { title: 'DASHBOARD', items: [{ to: '/', key: 'nav.dashboard' }, { to: '/noc', key: 'nav.noc' }] },
  { title: 'CUSTOMERS', items: [{ to: '/customers', key: 'nav.customers' }, { to: '/subscriptions', key: 'Subscriptions' }, { to: '/packages', key: 'Packages' }] },
  { title: 'BILLING', items: [{ to: '/invoices', key: 'billing.invoices' }, { to: '/payments', key: 'billing.payments' }] },
  { title: 'NETWORK', items: [{ to: '/devices', key: 'Devices' }, { to: '/connectors', key: 'Connectors' }, { to: '/topology', key: 'Topology' }, { to: '/lab', key: 'Connection Lab' }] },
  { title: 'SERVICES', items: [{ to: '/radius', key: 'RADIUS' }, { to: '/sessions', key: 'Sessions' }, { to: '/vouchers', key: 'Vouchers' }] },
  { title: 'FTTH', items: [{ to: '/olt', key: 'OLT' }, { to: '/onu', key: 'ONU/ONT' }] },
  { title: 'MONITORING', items: [{ to: '/alerts', key: 'Alerts' }, { to: '/events', key: 'Events' }, { to: '/incidents', key: 'Incidents' }] },
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
  return (
    <div className={dark ? 'app dark' : 'app'}>
      <aside className="sidebar">
        <div className="brand">Universal ISP</div>
        {groups.map(g => (
          <div key={g.title}><div className="gtitle">{g.title}</div>
            {g.items.map(it => (
              <Link key={it.to} to={it.to} className={loc.pathname === it.to ? 'nav active' : 'nav'}>{ctx.t(it.key)}</Link>
            ))}
          </div>
        ))}
      </aside>
      <main className="main">
        <header className="topbar">
          <strong>{ctx.t('noc.title')}</strong>
          <span className="sp" />
          {!authed
            ? <Link to="/login">{ctx.t('auth.login')}</Link>
            : <button onClick={logout}>{ctx.t('auth.logout')}</button>}
          <button onClick={() => ctx.setLang(ctx.lang === 'en' ? 'id' : 'en')}>{ctx.lang === 'en' ? 'ID' : 'EN'}</button>
          <button onClick={() => setDark(!dark)}>{dark ? '☀' : '☾'}</button>
        </header>
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
          <Route path="/" element={<Page k="nav.dashboard" />} />
          <Route path="*" element={<Page k={loc.pathname} />} />
        </Routes>
      </main>
    </div>
  )
}

function App() {
  const [lang, setLang] = useState<Lang>('en')
  const t = (k: string) => (lang === 'id' ? (id[k] ?? en[k] ?? k) : (en[k] ?? k))
  return (
    <LangCtx.Provider value={{ lang, setLang, t }}>
      <HashRouter><Shell /></HashRouter>
    </LangCtx.Provider>
  )
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>)
