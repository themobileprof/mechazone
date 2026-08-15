import type { HistoryResponse, Resolution, Session } from './types'

const shopId = import.meta.env.VITE_SHOP_ID ?? '00000000-0000-4000-8000-000000000001'
const technicianId = import.meta.env.VITE_TECHNICIAN_ID ?? '00000000-0000-4000-8000-000000000002'

export const identity = { shopId, technicianId }

export async function fetchHistory(vin: string): Promise<HistoryResponse> {
  const res = await fetch(`/api/v1/vehicles/${vin}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json() as Promise<HistoryResponse>
}

export async function decodeVin(vin: string): Promise<void> {
  await fetch(`/api/v1/vehicles/${vin}/decode`, { method: 'POST' })
}

export async function lookupDtc(code: string): Promise<{ code: string; title: string; cloud_ai_reserved?: boolean }> {
  const res = await fetch(`/api/v1/dtcs/${code}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json() as Promise<{ code: string; title: string; cloud_ai_reserved?: boolean }>
}

export async function ingestSession(body: unknown): Promise<Session> {
  const res = await fetch('/api/v1/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json() as Promise<Session>
}

export async function closeoutSession(id: string, body: unknown): Promise<Resolution> {
  const res = await fetch(`/api/v1/sessions/${id}/closeout`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json() as Promise<Resolution>
}

export async function ledgerOnline(): Promise<boolean> {
  try {
    const res = await fetch('/healthz')
    return res.ok
  } catch {
    return false
  }
}
