/** Shared DTOs with the Go ledger and Python worker. Customer name/phone/plate are this shop's file on the VIN. */
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

export type BusCapture = {
  vin: string
  profile: string
  adapter_type: string
  host_os: string
  protocol: string
  make_hint?: string
  model_hint?: string
  year_hint?: number
  modules: {
    name: string
    tx_id: string
    rx_id: string
    family?: string
    confirmed: boolean
    reachable: boolean
    ever_reachable: boolean
    dtcs?: string[]
  }[]
  identity: { name: string; did: string; text: string }[]
  live: { name: string; did: string; unit?: string; value?: unknown }[]
  active_codes: string[]
  coverage?: unknown
  raw_hex_excerpt?: string
  scan_count: number
  first_seen_at: string
  last_seen_at: string
}

export type PlaybookCheck = {
  id: string
  vin: string
  fingerprint: string
  kind: string
  title: string
  detail: string
  status: 'open' | 'done' | 'ruled_out'
  note?: string
  updated_at: string
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
  customer?: { display_name: string; phone: string; plate: string }
  capture?: BusCapture
  checks?: PlaybookCheck[]
  jobs: Job[]
  sessions: Session[]
  resolutions: Resolution[]
}

export type JobImport = {
  source: string
  original_name: string
  content_type: string
  byte_size: number
  note: string
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
  adapter_type?: string
  protocol?: string
  import?: JobImport
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

export type HowToGuide = {
  id: string
  slug?: string
  title: string
  blurb: string
  warning: string
  body_html: string
  match_words: string[]
  published: boolean
  action_ids?: string[]
  created_at?: string
  updated_at?: string
}

export type PlaybookAction = {
  id: string
  fingerprint: string
  kind: string
  title: string
  tokens: string[]
  variants: string[]
  seen_count: number
  guide_ids?: string[]
  first_seen_at: string
  last_seen_at: string
}

export type PlaybookStep = {
  order: number
  kind: string
  title: string
  detail: string
  pass?: string
  fail?: string
  adapter: boolean
  figures?: string[]
  howto_ids?: string[]
}


export type PlaybookFigure = {
  id: string
  title: string
  page: number
  caption: string
  language: string
  image_url?: string
  ocr_text?: string
  kind?: string
}

export type PlaybookAsk = {
  answer: string
  gaps: string[]
  figures?: PlaybookFigure[]
  model?: string
  retrieved_chunks?: number
}

export type Playbook = {
  vin: string
  platform: string
  lookouts: { text: string; evidence: string[] }[]
  likely_causes: { title: string; probability: number; evidence: string[] }[]
  steps: PlaybookStep[]
  validation: string
  gaps: string[]
  model?: string
  first_seen: boolean
  circuit_classes?: { code: string; class: string; reason: string }[]
  network?: { reading: string; summary: string; live: number; dark: number }
  manual_figures?: PlaybookFigure[]
  manual?: WorkshopBook
  retrieved_chunks?: number
  checks?: PlaybookCheck[]
}

export type WorkshopBook = {
  id: string
  title: string
  kind?: string
  make: string
  model: string
  year_from: number
  year_to: number
  engine: string
  language: string
  chunks: number
  figures: number
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

export type ScanCoverage = {
  id: string
  depth: string
  gaps: string[]
}

export type DetectedAdapter = {
  id: string
  label: string
  vid_pid?: string | null
  capability: string
  present: boolean
  connectable: boolean
  recommended: boolean
  library?: string | null
  gap?: string | null
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
  coverage?: ScanCoverage
  identity?: { name: string; did: string; text: string }[]
  cleared_codes?: string[]
}

export type ClearDtcModule = {
  name: string
  tx_id: string
  rx_id: string
  reachable: boolean
  attempted: boolean
  cleared: boolean
  codes_before: string[]
  codes_after: string[]
  error?: string
  gap?: string
}

export type ClearDtcsResult = {
  service: string
  group: string
  codes_before: string[]
  codes_after: string[]
  modules: ClearDtcModule[]
  circuit_classes?: { code: string; class: string; reason: string }[]
  raw_hex_stream: string[]
  gaps: string[]
}

export type DidStream = {
  seconds: number
  module: string
  tx_id: string
  io_control: string
  samples: { t: number; values: Record<string, number | null> }[]
  gap?: string
}

export type QueuedJob = {
  id: string
  kind: 'session' | 'closeout' | 'customer' | 'capture' | 'check'
  path: string
  method?: 'POST' | 'PUT'
  body: unknown
  created_at: string
}
