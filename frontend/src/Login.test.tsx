import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { HashRouter } from 'react-router-dom'
import { Login } from './pages/Login'

describe('Login page', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
  })
  it('renders username/password fields and submit', () => {
    render(<HashRouter><Login /></HashRouter>)
    expect(screen.getByLabelText(/Username|Nama pengguna/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/Password|Kata sandi/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Sign in|Masuk/i })).toBeInTheDocument()
  })
  it('shows demo hint without hardcoding secrets in labels', () => {
    render(<HashRouter><Login /></HashRouter>)
    expect(screen.getByText(/admin \/ secret/i)).toBeInTheDocument()
  })
})
