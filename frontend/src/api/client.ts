export const token = () => localStorage.getItem('isp_token') || ''

export async function api(path: string, init?: RequestInit) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const t = token()
  if (t) headers.Authorization = `Bearer ${t}`
  const r = await fetch(path, { ...init, headers: { ...headers, ...(init?.headers as any) } })
  if (r.status === 401) {
    localStorage.removeItem('isp_token')
    location.hash = '#/login'
    throw new Error('unauthorized')
  }
  if (!r.ok) throw new Error(`${r.status} ${r.statusText}`)
  return r.json()
}
