import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { HashRouter } from 'react-router-dom'
import { NOC } from './pages/NOC'
import { Lab } from './pages/Lab'

describe('NOC page', () => {
  beforeEach(() => { localStorage.clear() })
  afterEach(() => { vi.unstubAllGlobals() })
  it('renders live cards from API data', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: true,
      json: async () => ({ online_devices: 7, offline_devices: 2, active_subscribers: 100, suspended_subscribers: 3, online_sessions: 80, critical_alerts: 1 }),
    })))
    render(<HashRouter><NOC /></HashRouter>)
    await waitFor(() => expect(screen.getByText('100')).toBeInTheDocument())
    expect(screen.getByText('Devices online')).toBeInTheDocument()
  })
  it('shows honest error state when API fails', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => { throw new Error('down') }))
    render(<HashRouter><NOC /></HashRouter>)
    await waitFor(() => expect(screen.getByText(/offline/i)).toBeInTheDocument())
  })
})

describe('Lab page', () => {
  it('renders vendor/connection form without crashing', () => {
    render(<HashRouter><Lab /></HashRouter>)
    expect(screen.getByRole('button', { name: /Test connection|Uji koneksi/i })).toBeInTheDocument()
  })
})
