export type Principal = {
  user_id: string
  email: string
  role: 'super_admin' | 'technician'
  technician_id?: string
  technician_name?: string
  shop_id?: string
  shop_name?: string
  freelancer: boolean
}

export type AccessRequest = {
  id: string
  applicant_name: string
  contact_email: string
  contact_phone: string
  shop_name: string
  city: string
  country: string
  kind: 'shop' | 'freelancer'
  note: string
  status: 'pending' | 'provisioned' | 'dismissed'
  created_at: string
  reviewed_at?: string
  already_queued?: boolean
}

export type Shop = {
  id: string
  name: string
  location_country: string
  location_city: string
  created_at: string
}

export type Technician = {
  id: string
  shop_id?: string
  shop_name?: string
  full_name: string
  email: string
  reputation_score: number
  freelancer: boolean
  created_at: string
}

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
  jobs: Job[]
  sessions: Session[]
  resolutions: Resolution[]
}

export type Job = {
  session_id: string
  created_at: string
  mileage_km: number
  technician_name: string
  technician_id: string
  outcome: string
  active_codes: string[]
  work: string
  parts_replaced: string[]
  verified_fix: boolean
  resolution_id?: string
  closeout_code?: string
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

export type Playbook = {
  vin: string
  platform: string
  lookouts: { text: string; evidence: string[] }[]
  likely_causes: { title: string; probability: number; evidence: string[] }[]
  steps: {
    order: number
    kind: string
    title: string
    detail: string
    pass?: string
    fail?: string
    adapter: boolean
    figures?: string[]
  }[]
  validation: string
  gaps: string[]
  model?: string
  first_seen: boolean
  circuit_classes?: { code: string; class: string; reason: string }[]
  network?: { reading: string; summary: string; live: number; dark: number }
  manual_figures?: { id: string; title: string; page: number; caption: string; language: string; image_url?: string; ocr_text?: string; kind?: string }[]
}

export type ScanModule = {
  name: string
  tx_id: string
  rx_id: string
  family?: string
  confirmed: boolean
  reachable: boolean
  dtcs?: string[]
  error?: string
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
  modules: ScanModule[]
  circuit_classes?: { code: string; class: string; reason: string }[]
  network?: { reading: string; summary: string; live: number; dark: number }
}

export type DidStream = {
  seconds: number
  module: string
  tx_id: string
  io_control: string
  samples: { t: number; values: Record<string, number | null> }[]
}

export type QueuedJob = {
  id: string
  kind: 'session' | 'closeout'
  path: string
  body: unknown
  created_at: string
}
