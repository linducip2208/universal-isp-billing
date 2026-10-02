import { describe, expect, it, vi, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { HashRouter } from 'react-router-dom'
import { Invoices } from './pages/Billing'
import { Connectors } from './pages/Connectors'

describe('Billing pages', () => {
  afterEach(() => { vi.unstubAllGlobals() })
  it('renders invoice rows from API', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: true,
      json: async () => ({ data: [{ id: 'inv-1', status: 'open', total: 150000 }], total: 1, page: 1, per_page: 15 }),
    })))
    render(<HashRouter><Invoices /></HashRouter>)
    await waitFor(() => expect(screen.getByText('inv-1')).toBeInTheDocument())
  })
  it('shows empty state when no invoices', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: true, json: async () => ({ data: [], total: 0, page: 1, per_page: 15 }),
    })))
    render(<HashRouter><Invoices /></HashRouter>)
    await waitFor(() => expect(screen.getByText('—')).toBeInTheDocument())
  })
})

describe('Connectors page', () => {
  afterEach(() => { vi.unstubAllGlobals() })
  it('renders connector identity cards without secrets', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: true,
      json: async () => ({ data: [{ Vendor: 'MikroTik', ProductFamily: 'RouterOS', Status: 'PARTIAL', ConnectionType: 'cli' }] }),
    })))
    render(<HashRouter><Connectors /></HashRouter>)
    await waitFor(() => expect(screen.getByText('MikroTik')).toBeInTheDocument())
    expect(screen.queryByText(/password|secret/i)).not.toBeInTheDocument()
  })
})
