import { describe, expect, it, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { HashRouter } from 'react-router-dom'
import { Customers } from './pages/Customers'

describe('Customers page', () => {
  afterEach(() => { vi.unstubAllGlobals() })
  it('renders create form and list', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: true, json: async () => ({ data: [], total: 0 }),
    })))
    render(<HashRouter><Customers /></HashRouter>)
    expect(screen.getByRole('button', { name: 'Add customer' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Search' })).toBeInTheDocument()
  })
})
