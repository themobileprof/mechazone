/** Open bay visits parked on this laptop. Not a closeout and not another shop’s file. */
import { readVinPlate } from './vinReadout'
import type { DidStream, HistoryResponse, Playbook, PlaybookCheck, ScanCoverage, ScanResult, Session } from './types'

export type StallTab = 'kit' | 'vehicle' | 'capture' | 'playbook' | 'close'
export type StallPage = 'job' | 'file'

export type JobStall = {
  vin: string
  savedAt: number
  page: StallPage
  jobTab: StallTab
  headline: string
  tabLabel: string
  scan: ScanResult | null
  session: Session | null
  playbook: Playbook | null
  checks: PlaybookCheck[]
  mileage: string
  outcome: 'success' | 'failed'
  rootCause: string
  parts: string
  wiggle: DidStream | null
  dtcTitles: Record<string, string>
  dtcClass: Record<string, string>
  preCoverage: ScanCoverage | null
  manualId: string
  manualLocked: boolean
  kitVinMissed: boolean
}

const MAX = 8
const TAB_LABEL: Record<StallTab, string> = {
  kit: 'KIT',
  vehicle: 'VEHICLE',
  capture: 'CAPTURE',
  playbook: 'PLAYBOOK',
  close: 'CLOSE',
}

export function stallScope(shopId?: string, technicianId?: string) {
  return `mechazone.stalls.v1.${shopId || technicianId || 'bay'}`
}

export function stallHeadline(vin: string, vehicle?: HistoryResponse['vehicle'] | null, scan?: ScanResult | null) {
  const plate = readVinPlate(vin)
  const year = vehicle?.manufacture_year || scan?.year || plate?.year || ''
  const make = (vehicle?.make && vehicle.make.toLowerCase() !== 'unknown' ? vehicle.make : '') || scan?.make || plate?.maker || ''
  const model = (vehicle?.model && vehicle.model.toLowerCase() !== 'unknown' ? vehicle.model : '') || scan?.model || plate?.classLine || ''
  const title = [year, make, model].filter(Boolean).join(' ').trim()
  return title || plate?.headline || vin
}

export function stallDirty(s: Pick<JobStall, 'scan' | 'session' | 'playbook' | 'rootCause' | 'parts' | 'checks' | 'mileage'>) {
  return Boolean(
    s.scan || s.session || s.playbook || s.checks.length || s.mileage.trim() || s.rootCause.trim() || s.parts.trim(),
  )
}

export function loadStalls(scope: string): JobStall[] {
  try {
    const raw = localStorage.getItem(scope)
    if (!raw) return []
    const rows = JSON.parse(raw) as JobStall[]
    return Array.isArray(rows) ? rows.filter((r) => r && typeof r.vin === 'string' && r.vin.length === 17) : []
  } catch {
    return []
  }
}

function writeStalls(scope: string, rows: JobStall[]) {
  localStorage.setItem(scope, JSON.stringify(rows.slice(0, MAX)))
}

export function upsertStall(scope: string, stall: JobStall): JobStall[] {
  const slim: JobStall = {
    ...stall,
    scan: stall.scan ? { ...stall.scan, raw_hex_stream: (stall.scan.raw_hex_stream ?? []).slice(-40) } : null,
  }
  const rest = loadStalls(scope).filter((r) => r.vin !== slim.vin)
  const next = [slim, ...rest].slice(0, MAX)
  writeStalls(scope, next)
  return next
}

export function dropStall(scope: string, vin: string): JobStall[] {
  const next = loadStalls(scope).filter((r) => r.vin !== vin)
  writeStalls(scope, next)
  return next
}

export function tabLabel(id: StallTab) {
  return TAB_LABEL[id]
}
