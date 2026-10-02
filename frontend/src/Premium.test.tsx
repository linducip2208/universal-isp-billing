import { describe, expect, it, vi, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { HashRouter } from 'react-router-dom'
import { Incidents, Customer360 } from './pages/Premium'

describe('Incidents page', () => {
  afterEach(() => { vi.unstubAllGlobals() })
  it('renders rows with ack/assign/resolve actions', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: true,
      json: async () => ({ data: [{ id: 'i1', title: 'OLT down', severity: 'critical', status: 'open' }], total: 1 }),
    })))
    render(<HashRouter><Incidents /></HashRouter>)
    await waitFor(() => expect(screen.getByText('OLT down')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Ack' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Resolve' })).toBeInTheDocument()
  })
})

describe('Customer360 page', () => {
  it('renders loader form', () => {
    render(<HashRouter><Customer360 /></HashRouter>)
    expect(screen.getByText('Customer 360')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Load' })).toBeInTheDocument()
  })
})
