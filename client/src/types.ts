export type HistoryResponse = {
  vehicle: {
    vin: string
    make: string
    model: string
    manufacture_year: number
    decode_source: string
    first_seen_at: string
  } | null
  first_seen: boolean
  sessions: Session[]
  resolutions: Resolution[]
}

export type Session = {
  id: string
  vin: string
  shop_id: string
  technician_id: string
  mileage_km: number
  adapter_type: string
  host_os: string
  protocol: string
  active_codes: string[]
  freeze_frame: Record<string, unknown> | null
  outcome: string
  created_at: string
}

export type Resolution = {
  id: string
  session_id: string
  vin: string
  diagnostic_trouble_code: string
  root_cause_explanation: string
  parts_replaced: string[]
  is_verified_fix: boolean
  created_at: string
}

export type ScanResult = {
  vin: string
  profile: string
  make: string
  model: string
  year: number
  adapter_type: string
  protocol: string
  active_codes: string[]
  live: { name: string; value: number; unit: string; did: string }[]
  freeze_frame: Record<string, number>
  raw_hex_stream: string[]
  modules: { name: string; tx_id: string; rx_id: string; reachable: boolean; error?: string }[]
}

export type QueuedJob = {
  id: string
  kind: 'session' | 'closeout'
  path: string
  body: unknown
  created_at: string
}
