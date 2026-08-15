import type { HistoryResponse, Principal, Resolution, Session, Shop, Technician } from './types'

async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    ...init,
    credentials: 'include',
    headers: {
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...init.headers,
    },
  })
  if (!res.ok) {
    let msg = await res.text()
    try {
      msg = (JSON.parse(msg) as { error?: string }).error ?? msg
    } catch {
      /* keep text */
    }
    throw new Error(msg || res.statusText)
  }
  return res.json() as Promise<T>
}

export function login(email: string, password: string) {
  return api<Principal>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export function logout() {
  return api<{ status: string }>('/api/v1/auth/logout', { method: 'POST' })
}

export function me() {
  return api<Principal>('/api/v1/auth/me')
}

export function listShops() {
  return api<Shop[]>('/api/v1/admin/shops')
}

export function createShop(body: { name: string; location_city: string; location_country: string }) {
  return api<Shop>('/api/v1/admin/shops', { method: 'POST', body: JSON.stringify(body) })
}

export function listTechnicians() {
  return api<Technician[]>('/api/v1/admin/technicians')
}

export function createTechnician(body: { full_name: string; email: string; password: string; shop_id: string }) {
  return api<Technician>('/api/v1/admin/technicians', { method: 'POST', body: JSON.stringify(body) })
}

export function fetchHistory(vin: string) {
  return api<HistoryResponse>(`/api/v1/vehicles/${vin}`)
}

export function decodeVin(vin: string) {
  return api<unknown>(`/api/v1/vehicles/${vin}/decode`, { method: 'POST' })
}

export function lookupDtc(code: string) {
  return api<{ code: string; title: string; cloud_ai_reserved?: boolean }>(`/api/v1/dtcs/${code}`)
}

export function ingestSession(body: unknown) {
  return api<Session>('/api/v1/sessions', { method: 'POST', body: JSON.stringify(body) })
}

export function closeoutSession(id: string, body: unknown) {
  return api<Resolution>(`/api/v1/sessions/${id}/closeout`, { method: 'POST', body: JSON.stringify(body) })
}

export async function ledgerOnline(): Promise<boolean> {
  try {
    const res = await fetch('/healthz')
    return res.ok
  } catch {
    return false
  }
}
