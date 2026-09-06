/** Shop floor: this shop's jobs, OpenPort or attached report, playbook, closeout. */
import { useCallback, useEffect, useMemo, useState } from 'react'
import { attachImportedReport, buildPlaybook, closeoutSession, decodeVin, fetchHistory, importedReportURL, ingestSession, ledgerOnline, listHowTos, listManuals, lookupDtc, logout, saveCustomer, upsertBusCapture, upsertPlaybookCheck, type PlaybookBody } from './api'
import { Logo } from './Brand'
import {
  AttachIcon, BookIcon, ChassisIcon, CheckIcon, ChevronLeftIcon, ChevronRightIcon,
  ClearCodesIcon, FolderIcon, IconBtn, LinkIcon, LockIcon, MeterIcon, DetailIcon, PrintIcon, RefreshIcon, SaveIcon, ScanIcon,
  SignOutIcon, StampIcon, Tip, WaveIcon, WrenchIcon,
} from './chrome'
import { AskModal } from './AskModal'
import { ClearCodesModal } from './ClearCodesModal'
import { ShopFilePrint } from './ShopFilePrint'
import { HowToModal, type HowToCard } from './HowToModal'
import { matchCatalog, matchHowTos } from './howto'
import { enqueue, flushQueue, pendingCount } from './queue'
import { ToastStack, useAutoDismiss, type Notice } from './toast'
import type { ClearDtcsResult, DetectedAdapter, DidStream, HistoryResponse, HowToGuide, Playbook, PlaybookCheck, PlaybookStep, Principal, ScanCoverage, ScanResult, Session, WorkshopBook } from './types'
import { worker } from './worker'
import { kitVinGap, probePreview } from './vinProbe'

const IMPORT_SOURCES = [
  { id: 'x431', label: 'Launch X431' },
  { id: 'autel', label: 'Autel' },
  { id: 'techstream', label: 'Techstream / GTS' },
  { id: 'forscan', label: 'FORScan' },
  { id: 'golo', label: 'Golo' },
  { id: 'snap_on', label: 'Snap-on' },
  { id: 'other', label: 'Other scanner' },
] as const

type BayPage = 'job' | 'file'
type JobTab = 'kit' | 'vehicle' | 'capture' | 'playbook' | 'close'

const JOB_TABS = [
  { id: 'kit', n: '01', label: 'KIT', hint: 'Plug the OpenPort, connect' },
  { id: 'vehicle', n: '02', label: 'VEHICLE', hint: 'Type the VIN — the kit rarely fills it' },
  { id: 'capture', n: '03', label: 'CAPTURE', hint: 'Deep scan or attach a report' },
  { id: 'playbook', n: '04', label: 'PLAYBOOK', hint: 'Tick tests, then rebuild from findings' },
  { id: 'close', n: '05', label: 'CLOSE', hint: 'Log the visit, write what fixed it' },
] as const

function parseHash(): { page: BayPage; tab: JobTab } {
  const h = (typeof window === 'undefined' ? '' : window.location.hash).replace(/^#/, '')
  if (h === 'file' || h.startsWith('file/')) return { page: 'file', tab: 'kit' }
  const tab = h.split('/')[1]
  if (JOB_TABS.some((t) => t.id === tab)) return { page: 'job', tab: tab as JobTab }
  return { page: 'job', tab: 'kit' }
}

function tabLockHint(id: JobTab): string {
  if (id === 'capture') return 'Connect the kit or type a 17-character VIN first.'
  if (id === 'playbook' || id === 'close') return 'Deep-scan or attach a report first.'
  return ''
}

function namedModel(name: string | undefined) {
  const m = (name || '').trim()
  return m !== '' && m.toLowerCase() !== 'unknown'
}

function hintsFrom(
  vehicle: HistoryResponse['vehicle'] | null | undefined,
  book: WorkshopBook | null,
  scan: ScanResult | null,
) {
  const fromDecode = namedModel(vehicle?.model)
  return {
    make: (fromDecode ? vehicle?.make : undefined) || book?.make || vehicle?.make || scan?.make,
    model: (fromDecode ? vehicle?.model : undefined) || book?.model || vehicle?.model || scan?.model,
    year: vehicle?.manufacture_year || scan?.year || book?.year_from || 0,
  }
}

function captureBody(result: ScanResult) {
  return {
    profile: result.profile,
    adapter_type: result.adapter_type,
    host_os: navigator.platform.toLowerCase().includes('win') ? 'windows' : 'linux',
    protocol: result.protocol,
    make_hint: result.make,
    model_hint: result.model,
    year_hint: result.year,
    modules: (result.modules ?? []).map((m) => ({
      name: m.name,
      tx_id: m.tx_id,
      rx_id: m.rx_id,
      family: m.family,
      confirmed: m.confirmed,
      reachable: m.reachable,
      ever_reachable: m.reachable,
      dtcs: m.dtcs,
    })),
    identity: result.identity ?? [],
    live: result.live ?? [],
    active_codes: result.active_codes ?? [],
    coverage: result.coverage ?? {},
    raw_hex_stream: result.raw_hex_stream ?? [],
  }
}

function applyClearToScan(from: ScanResult, result: ClearDtcsResult): ScanResult {
  const byName = new Map(result.modules.map((m) => [m.name, m]))
  return {
    ...from,
    active_codes: result.codes_after,
    circuit_classes: result.circuit_classes ?? from.circuit_classes,
    raw_hex_stream: [...(from.raw_hex_stream ?? []), ...(result.raw_hex_stream ?? [])],
    cleared_codes: result.codes_before,
    modules: from.modules.map((m) => {
      const hit = byName.get(m.name)
      if (!hit) return m
      const dtcs = hit.attempted || hit.cleared ? hit.codes_after : m.dtcs
      return { ...m, reachable: hit.reachable || m.reachable, dtcs }
    }),
  }
}

function checkFingerprint(kind: string, title: string) {
  const k = (kind || 'test').trim().toLowerCase() || 'test'
  return `${k}|${title.trim().toLowerCase()}`
}

function matchManualId(books: WorkshopBook[], vehicle: HistoryResponse['vehicle'], scan: ScanResult | null) {
  if (scan?.profile === 'avensis_3zr_fae') {
    return books.find((b) => b.model.toLowerCase() === 'avensis')?.id ?? ''
  }
  const make = (vehicle?.make || scan?.make || '').toLowerCase()
  const model = (vehicle?.model || scan?.model || '').toLowerCase()
  const year = vehicle?.manufacture_year || scan?.year || 0
  if (!make || !namedModel(model)) return ''
  const hits = books.filter((b) => b.make.toLowerCase() === make && b.model.toLowerCase() === model)
  if (year) {
    const inYear = hits.find((b) => b.year_from <= year && b.year_to >= year)
    if (inYear) return inYear.id
  }
  return hits[0]?.id ?? ''
}

export function Bay({ user, onLogout }: { user: Principal; onLogout: () => void }) {
  const [online, setOnline] = useState(false)
  const [queued, setQueued] = useState(pendingCount())
  const [adapter, setAdapter] = useState('mock')
  const [kits, setKits] = useState<DetectedAdapter[]>([])
  const [connected, setConnected] = useState(false)
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [vin, setVin] = useState('')
  const [history, setHistory] = useState<HistoryResponse | null>(null)
  const [scan, setScan] = useState<ScanResult | null>(null)
  const [session, setSession] = useState<Session | null>(null)
  const [dtcTitles, setDtcTitles] = useState<Record<string, string>>({})
  const [mileage, setMileage] = useState('')
  const [customerName, setCustomerName] = useState('')
  const [customerPhone, setCustomerPhone] = useState('')
  const [customerPlate, setCustomerPlate] = useState('')
  const [outcome, setOutcome] = useState<'success' | 'failed'>('success')
  const [rootCause, setRootCause] = useState('')
  const [parts, setParts] = useState('')
  const [playbook, setPlaybook] = useState<Playbook | null>(null)
  const [checks, setChecks] = useState<PlaybookCheck[]>([])
  const [wiggle, setWiggle] = useState<DidStream | null>(null)
  const [dtcClass, setDtcClass] = useState<Record<string, string>>({})
  const [preCoverage, setPreCoverage] = useState<ScanCoverage | null>(null)
  const [importSource, setImportSource] = useState('x431')
  const [importCodes, setImportCodes] = useState('')
  const [importNote, setImportNote] = useState('')
  const [importFile, setImportFile] = useState<File | null>(null)
  const [manuals, setManuals] = useState<WorkshopBook[]>([])
  const [manualId, setManualId] = useState('')
  const [manualLocked, setManualLocked] = useState(false)
  const [notices, setNotices] = useState<Notice[]>([])
  const [page, setPage] = useState<BayPage>(() => parseHash().page)
  const [jobTab, setJobTab] = useState<JobTab>(() => parseHash().tab)
  const [lockHint, setLockHint] = useState<string | null>(null)
  const [kitVinMissed, setKitVinMissed] = useState(false)
  const [howTo, setHowTo] = useState<HowToCard[] | null>(null)
  const [howToCatalog, setHowToCatalog] = useState<HowToGuide[]>([])
  const [askStep, setAskStep] = useState<PlaybookStep | null>(null)
  const [confirmClear, setConfirmClear] = useState(false)

  const dismissNotice = useCallback((id: number) => {
    setNotices((rows) => rows.filter((n) => n.id !== id))
  }, [])
  useAutoDismiss(notices, dismissNotice)

  function pushNotice(kind: Notice['kind'], title: string, detail?: string) {
    const id = Date.now()
    setNotices((rows) => {
      const cleared = kind === 'busy' ? rows.filter((r) => r.kind !== 'busy') : rows.filter((r) => r.kind !== 'busy')
      return [...cleared, { id, kind, title, detail }]
    })
  }

  useEffect(() => {
    const tick = async () => {
      const ok = await ledgerOnline()
      setOnline(ok)
      if (ok) {
        await flushQueue()
        setQueued(pendingCount())
      }
    }
    void tick()
    const id = window.setInterval(() => void tick(), 8000)
    return () => window.clearInterval(id)
  }, [])

  useEffect(() => {
    if (!online) return
    void listManuals()
      .then(setManuals)
      .catch(() => setManuals([]))
    void listHowTos()
      .then(setHowToCatalog)
      .catch(() => setHowToCatalog([]))
  }, [online])

  async function refreshKits() {
    try {
      const d = await worker.detect()
      setKits(d.devices)
      setAdapter((current) => {
        const still = d.devices.some((x) => x.id === current)
        return still ? current : d.recommended
      })
    } catch {
      setKits([
        { id: 'mock', label: 'Mock ECU (bench)', capability: 'uds_mock', present: true, connectable: true, recommended: true },
        { id: 'openport2_rev_e', label: 'OpenPort 2.0 Rev E', capability: 'uds_j2534', present: false, connectable: true, recommended: false },
      ])
    }
  }

  useEffect(() => {
    void refreshKits()
  }, [])

  useEffect(() => {
    const onHash = () => {
      const route = parseHash()
      setPage(route.page)
      if (route.page !== 'job') return
      const captureOk = vin.length === 17 || connected
      const logged = Boolean(scan || session)
      const unlocked = route.tab === 'kit' || route.tab === 'vehicle' || (route.tab === 'capture' && captureOk) || ((route.tab === 'playbook' || route.tab === 'close') && logged)
      if (unlocked) {
        setJobTab(route.tab)
        setLockHint(null)
      } else {
        setLockHint(tabLockHint(route.tab))
      }
    }
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [vin, connected, scan, session])

  const codes = scan?.active_codes ?? []
  const liveOpenPort = Boolean(scan && scan.adapter_type !== 'imported_report')
  const canClear = Boolean(
    connected && liveOpenPort && (codes.length > 0 || scan?.modules.some((m) => (m.dtcs?.length ?? 0) > 0)),
  )
  const clearing = busy === 'clear codes'
  const clearTip = !connected
    ? 'Connect the kit on the Kit tab first'
    : !scan
      ? 'Deep-scan first'
      : !liveOpenPort
        ? 'Attached reports cannot be cleared on this OpenPort'
        : !canClear
          ? 'No codes on reachable modules'
          : 'Clear DTCs on modules that answered and currently have codes'
  const classFor = (code: string) =>
    scan?.circuit_classes?.find((c) => c.code === code)?.class || dtcClass[code] || ''

  useEffect(() => {
    void Promise.all(codes.map(async (code) => {
      try {
        const d = await lookupDtc(code)
        return [code, d.title || (d.cloud_ai_reserved ? 'Manufacturer-specific — use history/network' : ''), d.circuit_class || ''] as const
      } catch {
        return [code, '', ''] as const
      }
    })).then((rows) => {
      setDtcTitles(Object.fromEntries(rows.map(([c, t]) => [c, t])))
      setDtcClass(Object.fromEntries(rows.map(([c, , k]) => [c, k])))
    })
  }, [codes.join('|')])

  useEffect(() => {
    if (manualLocked) return
    const id = matchManualId(manuals, history?.vehicle ?? null, scan)
    if (id) setManualId(id)
  }, [manuals, history?.vehicle, scan, manualLocked])

  useEffect(() => {
    const loaded = history?.vehicle?.vin
    if (loaded && vin !== loaded) {
      setCustomerName('')
      setCustomerPhone('')
      setCustomerPlate('')
    }
  }, [vin, history?.vehicle?.vin])

  const selectedKit = kits.find((k) => k.id === adapter)
  const coverage = scan?.coverage ?? preCoverage
  const selectedBook = manuals.find((m) => m.id === manualId) ?? null
  const decodeMissedBody = !namedModel(history?.vehicle?.model) && !namedModel(scan?.model)
  const kitGaps = (coverage?.gaps ?? []).filter(kitVinGap)
  const mapGaps = (coverage?.gaps ?? []).filter((g) => !kitVinGap(g))
  const preview = probePreview({
    vin,
    model: history?.vehicle?.model,
    bookModel: selectedBook?.model,
    profile: scan?.profile,
  })

  const platformHints = () => hintsFrom(history?.vehicle, selectedBook, scan)

  const nextStep = ((): JobTab => {
    if (!connected && vin.length !== 17) return 'kit'
    if (vin.length !== 17 && !history) return 'vehicle'
    if (decodeMissedBody && manuals.length > 0 && !manualId) return 'vehicle'
    if (!scan && !session) return 'capture'
    if (!playbook) return 'playbook'
    return 'close'
  })()

  function tabUnlocked(id: JobTab): boolean {
    if (id === 'kit' || id === 'vehicle') return true
    if (id === 'capture') return vin.length === 17 || connected
    return Boolean(scan || session)
  }

  function tabComplete(id: JobTab): boolean {
    if (id === 'kit') return connected || vin.length === 17
    if (id === 'vehicle') return vin.length === 17
    if (id === 'capture') return Boolean(scan || session)
    if (id === 'playbook') return Boolean(playbook) || !online
    return false
  }

  function goPage(next: BayPage) {
    setPage(next)
    setLockHint(null)
    const hash = next === 'file' ? '#file' : `#job/${jobTab}`
    if (window.location.hash !== hash) window.history.replaceState(null, '', hash)
  }

  function goTab(next: JobTab, opts?: { force?: boolean }) {
    if (!opts?.force && !tabUnlocked(next)) {
      setLockHint(tabLockHint(next))
      return
    }
    setLockHint(null)
    setJobTab(next)
    setPage('job')
    const hash = `#job/${next}`
    if (window.location.hash !== hash) window.history.replaceState(null, '', hash)
  }

  function goBack() {
    const i = JOB_TABS.findIndex((t) => t.id === jobTab)
    if (i > 0) goTab(JOB_TABS[i - 1].id)
  }

  function goNext() {
    const i = JOB_TABS.findIndex((t) => t.id === jobTab)
    const nxt = JOB_TABS[i + 1]
    if (!nxt) return
    if (!tabComplete(jobTab)) {
      setLockHint(jobTab === 'kit'
        ? 'Connect the kit, or type the VIN and continue.'
        : jobTab === 'vehicle'
          ? 'VIN must be 17 characters before capture.'
          : jobTab === 'capture'
            ? 'Deep-scan or attach a report before the playbook.'
            : 'Rebuild the playbook before you close.')
      return
    }
    goTab(nxt.id)
  }

  const vehicleLabel = useMemo(() => {
    if (history?.vehicle && namedModel(history.vehicle.model)) {
      return `${history.vehicle.manufacture_year || ''} ${history.vehicle.make} ${history.vehicle.model}`.trim()
    }
    if (history?.vehicle?.make && history.vehicle.make.toLowerCase() !== 'unknown') {
      return `${history.vehicle.make} — pick the workshop book for the body`
    }
    if (scan?.make || scan?.model) return `${scan.year || ''} ${scan.make} ${scan.model}`.trim()
    if (scan?.coverage?.depth === 'captured') return scan.profile.replaceAll('_', ' ')
    if (scan?.profile === 'generic_uds') return 'ISO 15765-4 probe — no captured OEM map'
    if (scan?.profile === 'toyota_common') return 'Toyota 11-bit probe'
    if (vin.length === 17) return probePreview({ vin, model: history?.vehicle?.model, bookModel: selectedBook?.model }).label
    if (kitVinMissed) return 'Kit did not answer VIN — type the 17 characters from the plate'
    if (connected) return 'Type the VIN from the plate — most ECUs do not answer F190'
    return 'Type the VIN, or connect the kit'
  }, [history, scan, connected, vin, kitVinMissed, selectedBook?.model])

  async function run(label: string, fn: () => Promise<void>, okTitle?: string, okDetail?: string) {
    setBusy(label)
    setError(null)
    pushNotice('busy', label.toUpperCase())
    try {
      await fn()
      pushNotice('ok', okTitle || label, okDetail)
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
      pushNotice('fault', label, msg)
    } finally {
      setBusy(null)
    }
  }

  async function adviseFromScan(result: ScanResult, loggedVin: string, hints = platformHints()) {
    const body = playbookPayload(result, null, hints)
    if (!body) throw new Error('a live scan is required')
    body.vin = result.vin || loggedVin
    const book = await buildPlaybook(body)
    setPlaybook(book)
    if (book.checks) setChecks(book.checks)
  }

  function playbookPayload(fromScan: ScanResult | null = scan, fromSession: Session | null = session, hints = platformHints()): PlaybookBody | null {
    const lang = navigator.language.slice(0, 2) || 'en'
    if (fromScan) {
      return {
        vin: fromScan.vin || vin,
        make: hints.make,
        model: hints.model,
        year: hints.year,
        engine_hint: fromScan.profile,
        active_codes: fromScan.active_codes ?? [],
        live: fromScan.live,
        modules: fromScan.modules,
        freeze_frame: fromScan.freeze_frame,
        adapter_type: fromScan.adapter_type,
        protocol: fromScan.protocol,
        language: lang,
        source_id: manualId || undefined,
      }
    }
    if (fromSession) {
      return {
        vin: fromSession.vin,
        session_id: fromSession.id,
        make: hints.make,
        model: hints.model,
        year: hints.year,
        active_codes: fromSession.active_codes ?? [],
        adapter_type: 'imported_report',
        protocol: 'file_import',
        language: lang,
        source_id: manualId || undefined,
      }
    }
    return null
  }

  function applyHistory(h: HistoryResponse) {
    setCustomerName(h.customer?.display_name ?? '')
    setCustomerPhone(h.customer?.phone ?? '')
    setCustomerPlate(h.customer?.plate ?? '')
    setHistory((prev) => ({
      ...h,
      capture: h.capture ?? (prev?.capture && (!h.vehicle || prev.capture.vin === h.vehicle.vin) ? prev.capture : undefined),
      checks: h.checks ?? (prev && (!h.vehicle || prev.vehicle?.vin === h.vehicle.vin) ? prev.checks : undefined),
    }))
    if (h.checks) setChecks(h.checks)
  }

  async function persistCustomer() {
    const v = vin.trim().toUpperCase()
    if (v.length !== 17) return
    const body = {
      display_name: customerName.trim(),
      phone: customerPhone.trim(),
      plate: customerPlate.trim(),
    }
    const empty = !body.display_name && !body.phone && !body.plate
    const editingThisVin = history?.vehicle?.vin === v && Boolean(history.customer)
    if (empty && !editingThisVin) return
    try {
      if (!online) {
        enqueue({ kind: 'customer', method: 'PUT', path: `/api/v1/vehicles/${v}/customer`, body })
        setQueued(pendingCount())
        return
      }
      if (history?.vehicle?.vin !== v) {
        await decodeVin(v).catch(() => undefined)
      }
      const saved = await saveCustomer(v, body)
      setCustomerName(saved.display_name)
      setCustomerPhone(saved.phone)
      setCustomerPlate(saved.plate)
      if (history?.vehicle?.vin === v) {
        setHistory({
          ...history,
          customer: empty ? undefined : saved,
        })
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
      throw err
    }
  }

  function persistBusCapture(result: ScanResult, v: string) {
    if (v.length !== 17) return
    if (!result.adapter_type || !result.protocol) return
    if (result.adapter_type === 'imported_report') return
    const path = `/api/v1/vehicles/${v}/capture`
    const body = captureBody(result)
    const queue = () => {
      enqueue({ kind: 'capture', method: 'PUT', path, body })
      setQueued(pendingCount())
    }
    if (!online) {
      queue()
      return
    }
    void upsertBusCapture(v, body)
      .then((saved) => {
        setHistory((h) => {
          if (saved.vin !== v) return h
          if (!h) {
            return { vehicle: null, first_seen: true, jobs: [], sessions: [], resolutions: [], capture: saved }
          }
          if (h.vehicle && h.vehicle.vin !== saved.vin) return h
          return { ...h, capture: saved }
        })
      })
      .catch(() => {
        queue()
      })
  }

  function persistCheck(step: Playbook['steps'][number], status: PlaybookCheck['status'], note: string) {
    const v = (scan?.vin || vin).trim().toUpperCase()
    if (v.length !== 17) return
    const body = {
      fingerprint: checkFingerprint(step.kind, step.title),
      kind: step.kind,
      title: step.title,
      detail: step.detail,
      status,
      note,
    }
    const path = `/api/v1/vehicles/${v}/checks`
    const apply = (saved: PlaybookCheck) => {
      setChecks((rows) => {
        const rest = rows.filter((c) => c.fingerprint !== saved.fingerprint)
        return [saved, ...rest]
      })
    }
    const queue = () => {
      enqueue({ kind: 'check', method: 'PUT', path, body })
      setQueued(pendingCount())
      apply({
        id: body.fingerprint,
        vin: v,
        fingerprint: body.fingerprint,
        kind: body.kind,
        title: body.title,
        detail: body.detail,
        status,
        note,
        updated_at: new Date().toISOString(),
      })
    }
    if (!online) {
      queue()
      return
    }
    void upsertPlaybookCheck(v, body).then(apply).catch(queue)
  }

  async function loadTypedVin() {
    const v = vin.trim().toUpperCase()
    if (v.length !== 17) throw new Error('VIN must be 17 characters')
    await decodeVin(v).catch(() => undefined)
    applyHistory(await fetchHistory(v))
  }

  useEffect(() => {
    if (!online || vin.length !== 17) return
    if (history?.vehicle?.vin === vin) return
    const id = window.setTimeout(() => {
      void loadTypedVin().catch((err) => {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
      })
    }, 400)
    return () => window.clearTimeout(id)
  }, [vin, online, history?.vehicle?.vin])

  async function readVinFromKit() {
    const r = await worker.identify(vin.length === 17 ? vin : undefined)
    if (r.vin) {
      setKitVinMissed(false)
      if (r.vin !== vin) {
        setScan(null)
        setSession(null)
        setPlaybook(null)
      }
      setVin(r.vin)
    } else {
      setKitVinMissed(true)
    }
    const cov = r.coverage
    if (!r.vin) {
      const gaps = [...(cov?.gaps ?? [])]
      if (!gaps.some((g) => kitVinGap(g))) {
        gaps.unshift('Kit VIN (DID F190) stayed dark — type the 17 characters, then deep-scan. Modules still answer.')
      }
      setPreCoverage({
        id: cov?.id || r.profile || 'generic_uds',
        depth: cov?.depth || 'iso_15765_4',
        gaps,
      })
      return
    }
    setPreCoverage(cov ?? null)
    if (online) {
      await decodeVin(r.vin).catch(() => undefined)
      applyHistory(await fetchHistory(r.vin))
    } else {
      applyHistory({ vehicle: null, first_seen: true, jobs: [], sessions: [], resolutions: [] })
    }
  }

  async function adviseFromImport(sess: Session, typedCodes: string[]) {
    const body = playbookPayload(null, sess)
    if (!body) throw new Error('a live scan is required')
    body.active_codes = typedCodes.length ? typedCodes : sess.active_codes
    const book = await buildPlaybook(body)
    setPlaybook(book)
    if (book.checks) setChecks(book.checks)
  }

  const affiliation = user.freelancer ? 'Freelancer' : (user.shop_name || 'Shop')
  const tabIndex = JOB_TABS.findIndex((t) => t.id === jobTab)
  const nextTab = JOB_TABS[tabIndex + 1]
  const canAdvance = tabComplete(jobTab) && Boolean(nextTab) && tabUnlocked(nextTab.id)

  async function connectKit() {
    const r = await worker.connectAdapter(adapter)
    setConnected(r.connected)
    if (r.connected) {
      await readVinFromKit()
      goTab('vehicle')
    }
  }

  async function deepScan() {
    const typed = vin.trim().toUpperCase()
    let vehicle = history?.vehicle ?? null
    if (typed.length === 17 && online) {
      await decodeVin(typed).catch(() => undefined)
      try {
        const h = await fetchHistory(typed)
        applyHistory(h)
        vehicle = h.vehicle
      } catch {
        /* scan still runs — playbook uses whatever hints we have */
      }
    }
    const hints = hintsFrom(vehicle, selectedBook, scan)
    const result = await worker.scan({
      vin: typed || undefined,
      make: hints.make,
      model: hints.model,
      year: hints.year,
    })
    setScan(result)
    setPlaybook(null)
    setWiggle(null)
    const v = (result.vin || typed).trim().toUpperCase()
    if (result.vin && result.vin !== vin) {
      setVin(result.vin)
    }
    persistBusCapture(result, v)
    setConfirmClear(false)
    goTab('playbook')
    if (online && v.length === 17) {
      if (!vehicle || vehicle.vin !== v) {
        try {
          applyHistory(await fetchHistory(v))
        } catch {
          /* already scanned */
        }
      }
      try {
        await adviseFromScan(result, v, hintsFrom(vehicle, selectedBook, result))
      } catch {
        /* playbook optional — scan is on the worker; rebuild when the ledger answers */
      }
    }
  }

  async function clearCodes() {
    if (!scan) return
    const result = await worker.clearDtcs()
    const next = applyClearToScan(scan, result)
    setScan(next)
    persistBusCapture(next, (next.vin || vin).trim().toUpperCase())
    const anyCleared = result.modules.some((m) => m.cleared)
    if (!anyCleared) {
      throw new Error(result.gaps.join(' ') || 'ECU rejected $14')
    }
    setConfirmClear(false)
    if (result.gaps.length) {
      pushNotice('fault', result.codes_after.length ? 'Clear incomplete' : 'Cleared with gaps', result.gaps.join(' '))
    }
  }

  async function attachReport() {
    if (!importFile) return
    const form = new FormData()
    form.append('file', importFile)
    form.append('source', importSource)
    form.append('codes', importCodes)
    form.append('note', importNote)
    form.append('mileage_km', mileage || '0')
    form.append('host_os', navigator.platform.toLowerCase().includes('win') ? 'windows' : 'linux')
    if (history?.vehicle?.make) form.append('make_hint', history.vehicle.make)
    if (history?.vehicle?.model) form.append('model_hint', history.vehicle.model)
    const saved = await attachImportedReport(vin, form)
    setSession(saved.session)
    applyHistory(await fetchHistory(vin))
    setImportFile(null)
    const typed = saved.session.active_codes ?? []
    if (online && (typed.length > 0 || saved.session.id)) {
      try {
        await adviseFromImport(saved.session, typed)
      } catch {
        /* playbook optional — file is on the ledger */
      }
    }
    goTab('playbook')
  }

  function printShopFile() {
    const prev = document.title
    const shop = user.freelancer ? (user.technician_name || 'Freelancer') : (user.shop_name || 'Shop')
    document.title = `${shop} ${vin}`
    const restore = () => {
      document.title = prev
      window.removeEventListener('afterprint', restore)
    }
    window.addEventListener('afterprint', restore)
    window.print()
  }

  const canPrintFile = vin.length === 17 && Boolean(history)

  function guidesForStep(st: PlaybookStep): HowToCard[] {
    if (howToCatalog.length > 0) {
      return matchCatalog(st.title, st.detail, st.kind, howToCatalog, st.howto_ids)
    }
    return matchHowTos(st.title, st.detail, st.kind)
  }

  return (
    <>
    <div className="no-print relative min-h-svh px-5 py-4 md:px-8">
      <div className="grain" />
      <header className="mb-6 flex flex-wrap items-center justify-between gap-4 border-b border-brass/30 pb-4">
        <div className="flex items-center gap-4">
          <Logo className="h-14 w-auto md:h-16" />
          <div className="bay-pages" role="tablist" aria-label="Bay pages">
            <button
              type="button"
              role="tab"
              aria-selected={page === 'job'}
              className={`bay-page ${page === 'job' ? 'is-active' : ''}`}
              onClick={() => goPage('job')}
            >
              <WrenchIcon /> JOB
            </button>
            <Tip label={vin.length === 17 ? 'This shop’s jobs on this VIN' : 'Load a VIN on Job first'}>
              <button
                type="button"
                role="tab"
                aria-selected={page === 'file'}
                className={`bay-page ${page === 'file' ? 'is-active' : ''}`}
                onClick={() => goPage('file')}
              >
                <FolderIcon /> FILE
              </button>
            </Tip>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-3 font-mono text-xs">
          <Tip label={online ? 'Ledger reachable — jobs sync' : 'Offline — work queues until reconnect'}>
            <span className="inline-flex items-center gap-2">
              <span className={`h-2.5 w-2.5 rounded-full ${online ? 'bg-ok' : 'bg-fault'}`} />
              <span>{online ? 'LEDGER LIVE' : 'OFFLINE'}</span>
            </span>
          </Tip>
          <Tip label={queued ? `${queued} upload(s) waiting` : 'Nothing queued'}>
            <span className="text-steel">Q {queued}</span>
          </Tip>
          <span className="text-brass">{user.technician_name || user.email} · {affiliation}</span>
          <IconBtn tip="Sign out of this bay" onClick={() => void logout().then(onLogout)}>
            <SignOutIcon />
          </IconBtn>
        </div>
      </header>

      {page === 'job' && (
        <nav className="job-rail" aria-label="Job flow">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="job-tabs" role="tablist">
              {JOB_TABS.map((step) => {
                const unlocked = tabUnlocked(step.id)
                const done = tabComplete(step.id)
                const active = jobTab === step.id
                const recommended = nextStep === step.id
                const tip = unlocked
                  ? (done && !active ? `${step.hint} — done` : step.hint)
                  : tabLockHint(step.id)
                return (
                  <Tip key={step.id} label={tip}>
                    <button
                      type="button"
                      role="tab"
                      aria-selected={active}
                      aria-disabled={!unlocked}
                      className={`job-tab ${active ? 'is-active' : ''} ${recommended && !active ? 'is-next' : ''} ${unlocked ? '' : 'is-locked'}`}
                      onClick={() => goTab(step.id)}
                    >
                      {unlocked ? (done ? <CheckIcon /> : null) : <LockIcon />}
                      <span className="font-mono text-[10px] tracking-[0.28em]">{step.n}</span>
                      <span className="font-semibold tracking-wide">{step.label}</span>
                    </button>
                  </Tip>
                )
              })}
            </div>
            <div className="flex items-center gap-2">
              <IconBtn tip="Previous step" label="BACK" disabled={tabIndex === 0} onClick={goBack}>
                <ChevronLeftIcon />
              </IconBtn>
              {nextTab && (
                <IconBtn
                  tip={canAdvance ? `Next: ${nextTab.label}` : (lockHint || `Finish ${JOB_TABS[tabIndex].label} first`)}
                  label="NEXT"
                  tone="brass"
                  disabled={!canAdvance}
                  onClick={goNext}
                >
                  <ChevronRightIcon />
                </IconBtn>
              )}
            </div>
          </div>
          <p className="mt-2 max-w-xl text-xs text-steel">{JOB_TABS[tabIndex]?.hint}</p>
          {(lockHint || busy || error) && (
            <p className={`mt-1 font-mono text-xs tracking-wide ${error ? 'text-fault' : lockHint ? 'text-brass' : 'text-brass'}`}>
              {error || lockHint || `${busy?.toUpperCase()}…`}
            </p>
          )}
        </nav>
      )}

      {page === 'file' && (
        <section className="rounded-sm border border-brass/20 bg-panel p-5">
          <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 className="font-mono text-sm tracking-[0.25em] text-brass">THIS SHOP'S WORK</h2>
              <p className="mt-1 text-sm text-steel">{vin.length === 17 ? vin : 'No VIN loaded — switch to Job and type or read one.'}</p>
              {canPrintFile && (
                <p className="mt-1 text-xs text-steel">PRINT opens this shop’s jobs. Save as PDF in the browser dialog. Phone numbers stay off the sheet.</p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <IconBtn
                tip={canPrintFile ? 'Print or save this shop’s jobs as PDF' : 'Load a 17-character VIN first'}
                label="PRINT"
                disabled={!canPrintFile}
                onClick={printShopFile}
              >
                <PrintIcon />
              </IconBtn>
              <IconBtn tip="Back to the current visit" label="JOB" onClick={() => goPage('job')}>
                <WrenchIcon />
              </IconBtn>
            </div>
          </div>
          {!history && <p className="text-steel">Read VIN from the kit, or type the 17-character VIN and load this shop's jobs.</p>}
          {history?.capture && (
            <div className="mb-4 border border-brass/20 bg-oil/50 p-3">
              <p className="font-mono text-[11px] tracking-[0.25em] text-brass">BUS MAP · NOT A JOB</p>
              <p className="mt-1 text-sm text-steel">
                {history.capture.profile || 'no profile'} · {history.capture.adapter_type} · {history.capture.protocol.replaceAll('_', ' ')}
                {' · '}
                {history.capture.scan_count === 1 ? '1 scan' : `${history.capture.scan_count} scans`}
              </p>
              <p className="mt-1 font-mono text-xs text-steel">
                {history.capture.modules.filter((m) => m.ever_reachable).length
                  ? history.capture.modules.filter((m) => m.ever_reachable).map((m) => `${m.name} ${m.tx_id}${m.reachable ? '' : ' (dark now)'}`).join(' · ')
                  : 'No nodes have answered yet'}
              </p>
              {history.capture.active_codes.length > 0 && (
                <p className="mt-1 font-mono text-xs text-steel">Last codes {history.capture.active_codes.join('  ')}</p>
              )}
              <p className="mt-2 text-xs text-steel">Observed on this shop’s cable. Log and close a job to record the work done.</p>
            </div>
          )}
          {history?.first_seen && !history.capture && <p className="text-steel">This shop has not worked this vehicle yet. Close the job to start the file.</p>}
          {history?.first_seen && history.capture && <p className="mb-3 text-steel">No job closed yet. The bus map above is from scans, not a closeout.</p>}
          <ol className="space-y-3">
            {(history?.jobs ?? []).map((job) => (
              <li key={job.session_id} className={`border-l-2 pl-3 ${job.verified_fix ? 'border-ok' : 'border-steel/50'}`}>
                <p className="font-mono text-xs text-steel">
                  {new Date(job.created_at).toLocaleString()} · {job.technician_name || 'tech'} · {job.mileage_km} km · {job.outcome}
                  {job.import ? ` · IMPORTED · ${job.import.source.toUpperCase()}` : job.adapter_type ? ` · ${job.adapter_type}` : ''}
                  {job.verified_fix ? ' · CLOSED' : ''}
                </p>
                {job.work ? (
                  <p>{job.work}</p>
                ) : job.import?.note ? (
                  <p className="text-sm">{job.import.note}</p>
                ) : (
                  <p className="text-sm text-steel">{job.import ? 'Report attached — close the job to record the work done.' : 'Scan logged — close the job to record the work done.'}</p>
                )}
                {job.parts_replaced.length > 0 && (
                  <p className="text-sm text-steel">Parts: {job.parts_replaced.join(', ')}</p>
                )}
                {(job.closeout_code || job.active_codes.length > 0) && (
                  <p className="font-mono text-xs text-steel">
                    {job.closeout_code || job.active_codes.join('  ')}
                  </p>
                )}
                {job.import && (
                  <a className="mt-1 inline-block font-mono text-[11px] tracking-widest text-brass underline-offset-2 hover:underline" href={importedReportURL(job.session_id)} target="_blank" rel="noreferrer">
                    OPEN FILE · {job.import.original_name}
                  </a>
                )}
              </li>
            ))}
          </ol>
        </section>
      )}

      {page === 'job' && jobTab === 'kit' && (
        <section className="mb-6 rounded-sm border border-brass/25 bg-panel/80 p-5">
          <p className="font-mono text-[11px] tracking-[0.3em] text-steel">KIT</p>
          <p className="mt-1 max-w-2xl text-sm text-steel">Plug the OpenPort and connect. Most cars do not answer VIN DID F190. Type the 17 characters on Vehicle, then deep-scan — that is how module readings and the playbook are chosen.</p>
          <div className="mt-4 grid gap-3 md:grid-cols-[1fr_auto_auto_auto] md:items-end">
            <label className="block">
              <span className="font-mono text-[11px] tracking-widest text-steel">ADAPTER</span>
              <select
                className="mt-1 w-full border border-steel/40 bg-oil px-3 py-3 text-paper"
                value={adapter}
                onChange={(e) => setAdapter(e.target.value)}
              >
                {(kits.length ? kits : [
                  { id: 'mock', label: 'Mock ECU (bench)', connectable: true },
                  { id: 'openport2_rev_e', label: 'OpenPort 2.0 Rev E', connectable: true },
                ]).map((k) => (
                  <option key={k.id} value={k.id}>
                    {k.label}{'present' in k && k.present && k.id !== 'mock' ? ' · seen' : ''}{k.connectable ? '' : ' · detect only'}
                  </option>
                ))}
              </select>
            </label>
            <IconBtn tip="Re-detect USB adapters on this laptop" label="KITS" onClick={() => run('detect', refreshKits)}>
              <RefreshIcon />
            </IconBtn>
            <IconBtn
              tip={connected ? 'Drop and reconnect the Pass-Thru' : 'Open the Pass-Thru. VIN from the ECU is optional.'}
              label={connected ? 'RECONNECT' : 'CONNECT'}
              tone="brass"
              disabled={selectedKit?.connectable === false}
              onClick={() => run('connect', connectKit, 'Kit connected', 'Type the VIN on Vehicle if the field is empty — most ECUs do not answer F190. Then deep-scan.')}
            >
              <LinkIcon />
            </IconBtn>
            <IconBtn
              tip="Ask the ECU for VIN DID F190 — often stays dark; typing it is the normal path"
              label="VIN"
              disabled={!connected}
              onClick={() => run('read VIN', readVinFromKit, 'VIN from kit', 'If the field is still empty, type the plate VIN — F190 staying dark is normal.')}
            >
              <ChassisIcon />
            </IconBtn>
          </div>
          {selectedKit?.gap && <p className="mt-3 text-sm text-steel">{selectedKit.gap}</p>}
          <p className="mt-4 font-mono text-xs text-steel">{connected ? (kitVinMissed || vin.length !== 17 ? 'Kit is up. Type the VIN on Vehicle — then Capture.' : 'Kit is up. Confirm the VIN on Vehicle, then Capture.') : 'No kit yet — you can still type a VIN on Vehicle.'}</p>
        </section>
      )}

      {page === 'job' && jobTab === 'vehicle' && (
        <section className="mb-6 rounded-sm border border-brass/20 bg-panel p-5">
          <p className="font-mono text-[11px] tracking-[0.3em] text-steel">VEHICLE</p>
          <p className="mt-1 max-w-2xl text-sm text-steel">Type the VIN from the plate or windscreen. The kit almost never fills this field. Decode and this shop’s file load as soon as all 17 characters are in.</p>
          <div className="mt-1 flex flex-wrap items-end gap-3">
            <label className="min-w-[16rem] flex-1">
              <span className="sr-only">VIN</span>
              <input
                className="w-full border-b border-brass/40 bg-transparent font-mono text-3xl tracking-[0.18em] text-paper outline-none md:text-5xl"
                value={vin}
                spellCheck={false}
                autoCapitalize="characters"
                placeholder="———— — ——————"
                onChange={(e) => setVin(e.target.value.toUpperCase().replace(/[^A-HJ-NPR-Z0-9]/g, '').slice(0, 17))}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && vin.length === 17 && online) void run('vin', loadTypedVin)
                }}
              />
            </label>
            <IconBtn
              tip="Decode this VIN and open this shop’s jobs on File"
              label="LOAD"
              disabled={vin.length !== 17 || !online}
              onClick={() => run('vin', loadTypedVin, 'VIN loaded', 'This shop’s jobs are on the File page.')}
            >
              <ChassisIcon />
            </IconBtn>
          </div>
          <div className="mt-2 flex flex-wrap gap-4 text-sm text-steel">
            <span>{vehicleLabel}</span>
            {history && (
              <span className={history.first_seen ? 'text-brass' : 'text-ok'}>
                {history.first_seen ? 'FIRST VISIT TO THIS SHOP' : `${history.jobs?.length ?? history.sessions.length} JOB(S) HERE`}
                {history.capture ? ` · BUS MAP ×${history.capture.scan_count}` : ''}
                {customerPlate ? ` · ${customerPlate}` : ''}
              </span>
            )}
            {coverage && (
              <span className="font-mono text-xs text-brass">{coverage.id} · {coverage.depth.replaceAll('_', ' ')}</span>
            )}
            {vin.length === 17 && (
              <span className="font-mono text-xs text-brass">{preview.label}</span>
            )}
          </div>
          {(kitVinMissed || kitGaps.length > 0) && vin.length !== 17 && (
            <div className="mt-3 border border-brass/30 bg-oil/60 px-3 py-3">
              <p className="font-mono text-[11px] tracking-widest text-brass">KIT DID NOT READ THE VIN</p>
              <p className="mt-2 text-sm text-steel">That is a dark DID F190, not a dead scan. Type the 17 characters above, then go to Capture.</p>
            </div>
          )}
          {mapGaps.length > 0 && (
            <div className="mt-3 border border-brass/30 bg-oil/60 px-3 py-3">
              <p className="font-mono text-[11px] tracking-widest text-brass">WHAT THIS MAP STILL NEEDS</p>
              <ul className="mt-2 space-y-1 text-sm text-steel">
                {mapGaps.map((g) => <li key={g}>{g}</li>)}
              </ul>
            </div>
          )}
          <div className="mt-4 grid gap-3 border border-brass/20 bg-oil/50 p-3 md:grid-cols-[1fr_auto] md:items-end">
            <label className="block">
              <span className="font-mono text-[11px] tracking-widest text-brass">WORKSHOP BOOK</span>
              <select
                className="mt-1 w-full border border-steel/40 bg-oil px-3 py-3 text-paper"
                value={manualId}
                onChange={(e) => {
                  setManualLocked(true)
                  setManualId(e.target.value)
                }}
              >
                <option value="">
                  {manuals.length === 0
                    ? 'No ingested manuals on this ledger'
                    : decodeMissedBody
                      ? 'Decode did not name the body — pick the book'
                      : 'Auto from VIN / leave unset'}
                </option>
                {manuals.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.make} {m.model} {m.year_from}–{m.year_to} · {m.title} ({m.language})
                  </option>
                ))}
              </select>
            </label>
            <p className="max-w-md text-sm text-steel">
              {selectedBook
                ? `${selectedBook.chunks.toLocaleString()} pages · ${selectedBook.figures.toLocaleString()} figures on file. Capture uses this map; the playbook cites this book.`
                : 'Avensis T27 RM is ingested. If vPIC only said Toyota, pick that book here so AI is not flying blind.'}
            </p>
          </div>
          <div className="mt-4 grid gap-3 md:grid-cols-3">
            <label className="block">
              <span className="font-mono text-[11px] text-steel">CUSTOMER NAME</span>
              <input
                className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2"
                value={customerName}
                onChange={(e) => setCustomerName(e.target.value)}
                onBlur={() => void persistCustomer().catch((err) => {
                  const msg = err instanceof Error ? err.message : String(err)
                  pushNotice('fault', 'customer', msg)
                })}
                placeholder="This shop’s file — follows the login"
              />
            </label>
            <label className="block">
              <span className="font-mono text-[11px] text-steel">PHONE</span>
              <input
                className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2"
                value={customerPhone}
                onChange={(e) => setCustomerPhone(e.target.value)}
                onBlur={() => void persistCustomer().catch((err) => {
                  const msg = err instanceof Error ? err.message : String(err)
                  pushNotice('fault', 'customer', msg)
                })}
                placeholder="0803…"
              />
            </label>
            <label className="block">
              <span className="font-mono text-[11px] text-steel">PLATE</span>
              <input
                className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2 font-mono uppercase"
                value={customerPlate}
                onChange={(e) => setCustomerPlate(e.target.value.toUpperCase())}
                onBlur={() => void persistCustomer().catch((err) => {
                  const msg = err instanceof Error ? err.message : String(err)
                  pushNotice('fault', 'customer', msg)
                })}
                placeholder="ABC-123"
              />
            </label>
          </div>
          <p className="mt-2 max-w-2xl text-sm text-steel">
            Name, phone, and plate live on this shop’s ledger so a new laptop still has them. Other shops cannot read them. They never go to the playbook or the OpenPort worker.
          </p>
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <IconBtn
              tip="Save name, phone, and plate to this shop’s ledger, then continue to Capture"
              label="SAVE"
              disabled={vin.length !== 17 || Boolean(busy)}
              onClick={() => run('customer', async () => {
                await persistCustomer()
                goNext()
              }, 'Customer saved', 'Capture is next. This shop’s file, not this laptop.')}
            >
              <SaveIcon />
            </IconBtn>
            <IconBtn
              tip={canAdvance ? 'Continue to Capture without changing the customer file' : (lockHint || 'VIN must be 17 characters before capture.')}
              label="NEXT"
              tone="brass"
              disabled={!canAdvance || Boolean(busy)}
              onClick={goNext}
            >
              <ChevronRightIcon />
            </IconBtn>
          </div>
        </section>
      )}

      {page === 'job' && jobTab === 'capture' && (
        <div className="space-y-6">
          <div className="grid gap-6 xl:grid-cols-2">
            <section className="rounded-sm border border-brass/25 bg-panel p-5">
              <p className="font-mono text-[11px] tracking-[0.3em] text-brass">THIS OPENPORT</p>
              <h2 className="mt-1 font-poster text-2xl tracking-wide text-paper">DEEP SCAN</h2>
              <p className="mt-2 max-w-xl text-sm text-steel">
                {vin.length === 17
                  ? `${preview.label}. Modules that answer still return codes even when the kit never sent a VIN.`
                  : 'Type the 17-character VIN on Vehicle first. The kit rarely reads it; the typed VIN is what picks Toyota vs a generic 7E0–7E2 probe.'}
              </p>
              <div className="mt-4 flex flex-wrap items-center gap-3">
                <IconBtn
                  tip={!connected ? 'Connect the kit on the Kit tab first' : vin.length !== 17 ? 'Scan anyway — without a VIN this is a generic 7E0–7E2 probe' : 'Scan modules, then open the playbook'}
                  label="DEEP SCAN"
                  tone="brass"
                  disabled={!connected}
                  onClick={() => run(online ? 'scan + playbook' : 'scan', deepScan, 'Deep scan complete', online ? 'Playbook fused this scan with this shop and the selected book.' : 'Ledger offline — rebuild the playbook when you reconnect.')}
                >
                  <ScanIcon />
                </IconBtn>
                <IconBtn
                  tip={clearTip}
                  label="CLEAR CODES"
                  disabled={!canClear || clearing}
                  onClick={() => setConfirmClear(true)}
                >
                  <ClearCodesIcon />
                </IconBtn>
              </div>
            </section>
            <section className="border border-dashed border-brass/50 bg-oil/70 p-5">
              <div className="mb-3 flex flex-wrap items-baseline justify-between gap-3">
                <div>
                  <p className="font-mono text-[11px] tracking-[0.3em] text-brass">NOT THIS OPENPORT</p>
                  <h2 className="mt-1 font-poster text-2xl tracking-wide text-paper">ATTACH REPORT</h2>
                </div>
                <span className="font-mono text-[11px] tracking-widest text-steel">LEDGER ONLINE ONLY</span>
              </div>
              <p className="text-sm text-steel">
                PDF, photo, or CSV from X431 / Autel / Techstream. Type the codes you see. We do not OCR the file.
                Strip customer name and plate from the file — those belong on Vehicle.
              </p>
              <div className="mt-4 grid gap-3">
                <label className="block">
                  <span className="font-mono text-[11px] tracking-widest text-steel">SOURCE</span>
                  <select className="mt-1 w-full border border-steel/40 bg-oil px-3 py-3 text-paper" value={importSource} onChange={(e) => setImportSource(e.target.value)}>
                    {IMPORT_SOURCES.map((s) => (
                      <option key={s.id} value={s.id}>{s.label}</option>
                    ))}
                  </select>
                </label>
                <label className="block">
                  <span className="font-mono text-[11px] tracking-widest text-steel">CODES ON THE REPORT</span>
                  <input
                    className="mt-1 w-full border border-steel/40 bg-oil px-3 py-3 font-mono text-paper"
                    value={importCodes}
                    onChange={(e) => setImportCodes(e.target.value.toUpperCase())}
                    placeholder="P0301 P0420 U0100"
                  />
                </label>
                <label className="block">
                  <span className="font-mono text-[11px] tracking-widest text-steel">MILEAGE KM</span>
                  <input className="mt-1 w-full border border-steel/40 bg-oil px-3 py-3 font-mono" value={mileage} onChange={(e) => setMileage(e.target.value)} placeholder="km" />
                </label>
                <label className="block">
                  <span className="font-mono text-[11px] tracking-widest text-steel">MECHANICAL NOTE (NO NAME / PHONE / PLATE)</span>
                  <input className="mt-1 w-full border border-steel/40 bg-oil px-3 py-3" value={importNote} onChange={(e) => setImportNote(e.target.value)} placeholder="Misfire under load, codes from X431 printout" />
                </label>
              </div>
              <div className="mt-4 flex flex-wrap items-center gap-3">
                <label className="min-h-12 cursor-pointer border border-brass/60 px-4 py-3 font-mono text-xs tracking-widest text-brass">
                  {importFile ? importFile.name : 'CHOOSE FILE'}
                  <input
                    className="sr-only"
                    type="file"
                    accept=".pdf,.png,.jpg,.jpeg,.webp,.txt,.csv,.json,application/pdf,image/jpeg,image/png,image/webp,text/plain,text/csv,application/json"
                    onChange={(e) => setImportFile(e.target.files?.[0] ?? null)}
                  />
                </label>
                <IconBtn
                  tip={!online ? 'Ledger must be online to attach' : vin.length !== 17 ? '17-character VIN required' : !importFile ? 'Choose a file first' : 'Attach this report to the VIN'}
                  label="ATTACH"
                  tone="brass"
                  disabled={!online || vin.length !== 17 || !importFile}
                  onClick={() => run('attach report', attachReport, 'Report attached', 'Playbook uses typed codes plus the selected workshop book.')}
                >
                  <AttachIcon />
                </IconBtn>
              </div>
            </section>
          </div>
          <section className="rounded-sm border border-brass/20 bg-panel p-5">
            <h2 className="mb-3 font-mono text-sm tracking-[0.25em] text-brass">LIVE MODULES</h2>
            {!scan && <p className="text-steel">Deep scan uses the typed VIN: a captured platform map when we have one, otherwise a Toyota 11-bit probe or ISO 15765-4 (7E0–7E2). Dark means no UDS answer — not a generic PID miss, and not a missing VIN from the kit.</p>}
            {scan?.network && (
              <p className="mb-3 text-sm text-steel">{scan.network.summary}</p>
            )}
            <div className="mb-4 flex flex-wrap gap-2">
              {scan?.modules.map((m) => (
                <span key={m.name} className={`border px-2 py-1 font-mono text-xs ${m.reachable ? 'border-ok text-ok' : 'border-fault text-fault'}`}>
                  {m.name} {m.tx_id}{m.confirmed ? '' : ' · probe'}
                </span>
              ))}
            </div>
            <ul className="mb-4 space-y-2">
              {codes.map((code) => (
                <li key={code} className="flex justify-between gap-3 border border-fault/30 bg-fault/5 px-3 py-2">
                  <span className="font-mono text-fault">{code}</span>
                  <span className="text-right text-sm text-steel">
                    {classFor(code) && classFor(code) !== 'component' ? `${classFor(code).replaceAll('_', ' ')} · ` : ''}
                    {dtcTitles[code]}
                  </span>
                </li>
              ))}
            </ul>
            {scan?.cleared_codes && scan.cleared_codes.length > 0 && (
              <p className="mb-4 text-sm text-steel">
                Cleared this visit: <span className="font-mono text-paper">{scan.cleared_codes.join(' ')}</span>
                {codes.length === 0 ? ' — none remaining on the modules that answered.' : ''}
              </p>
            )}
            {scan?.identity && scan.identity.length > 0 && (
              <dl className="mb-3 grid grid-cols-2 gap-2 font-mono text-xs text-steel">
                {scan.identity.map((row) => (
                  <div key={row.did} className="border border-steel/20 px-3 py-2">
                    <dt>{row.name}</dt>
                    <dd className="text-paper">{row.text}</dd>
                  </div>
                ))}
              </dl>
            )}
            <dl className="grid grid-cols-2 gap-2 font-mono text-sm">
              {scan?.live.map((row) => (
                <div key={row.name} className="border border-steel/20 px-3 py-2">
                  <dt className="text-[11px] text-steel">{row.name}</dt>
                  <dd>{row.value} {row.unit}</dd>
                </div>
              ))}
            </dl>
            <div className="mt-4 flex flex-wrap items-center gap-3">
              <IconBtn
                tip="Stream ECM DIDs while you wiggle the harness"
                label="WIGGLE"
                disabled={!connected || !scan || !(scan.live?.length)}
                onClick={() => run('wiggle', async () => {
                  setWiggle(await worker.streamDids(6))
                })}
              >
                <WaveIcon />
              </IconBtn>
              <IconBtn
                tip={clearTip}
                label="CLEAR CODES"
                disabled={!canClear || clearing}
                onClick={() => setConfirmClear(true)}
              >
                <ClearCodesIcon />
              </IconBtn>
            </div>
            {wiggle && (
              <div className="mt-3 font-mono text-xs text-steel">
                <p>{wiggle.gap || `${wiggle.samples.length} samples · IO control ${wiggle.io_control.replaceAll('_', ' ')}`}</p>
                <ul className="mt-2 max-h-40 space-y-1 overflow-auto">
                  {wiggle.samples.map((s, i) => {
                    const keys = Object.keys(s.values)
                    return (
                      <li key={i}>
                        t={s.t}s {keys.slice(0, 3).map((k) => `${k} ${s.values[k] ?? '—'}`).join('  ')}
                      </li>
                    )
                  })}
                </ul>
              </div>
            )}
          </section>
        </div>
      )}

      {page === 'job' && jobTab === 'playbook' && (
        <section className="mb-6 rounded-sm border border-brass/20 bg-panel p-5">
          <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 className="font-mono text-sm tracking-[0.25em] text-brass">AI PLAYBOOK</h2>
              <p className="mt-1 text-sm text-steel">Tick what you ran or ruled out. Rebuild fuses those findings with the live scan so the next tests can move.</p>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <IconBtn
                tip={clearTip}
                label="CLEAR CODES"
                disabled={!canClear || clearing}
                onClick={() => setConfirmClear(true)}
              >
                <ClearCodesIcon />
              </IconBtn>
              <IconBtn
                tip={!online ? 'Ledger offline — reconnect to rebuild' : (!scan && !session) ? 'Capture a scan or report first' : 'Fuse this scan with this shop, the pinned book, and the checks you ticked'}
                label="REBUILD"
                tone="brass"
                disabled={!vin || !online || (!scan && !session)}
                onClick={() => run('playbook', async () => {
                  if (scan) {
                    await adviseFromScan(scan, vin)
                    return
                  }
                  if (session) await adviseFromImport(session, session.active_codes ?? [])
                }, 'Playbook ready', selectedBook ? `Cited ${selectedBook.title}` : 'Next tests use the checks you ticked.')}
              >
                <BookIcon />
              </IconBtn>
            </div>
          </div>
          {!playbook && <p className="text-steel">Deep scan or an attached report writes the playbook when the ledger is online. Offline, scan first and rebuild when you reconnect.</p>}
          {scan?.cleared_codes && scan.cleared_codes.length > 0 && (
            <p className="mt-3 text-sm text-steel">
              Cleared this visit: <span className="font-mono text-paper">{scan.cleared_codes.join(' ')}</span>
              {codes.length === 0 ? ' — none remaining on the modules that answered.' : ` · remaining ${codes.join(' ')}`}
            </p>
          )}
          {playbook && (
            <div className="space-y-4">
              {playbook.manual && (
                <p className="border border-brass/40 bg-brass/10 px-3 py-2 text-sm">
                  <span className="font-mono text-[11px] tracking-widest text-brass">BOOK </span>
                  {playbook.manual.title} · {playbook.manual.make} {playbook.manual.model} {playbook.manual.year_from}–{playbook.manual.year_to}
                  {playbook.retrieved_chunks ? ` · ${playbook.retrieved_chunks} retrieved pages` : ''}
                </p>
              )}
              <p className="font-mono text-xs text-steel">
                {playbook.platform || 'platform unknown'}
                {playbook.first_seen ? ' · first visit to this shop' : ''}
                {playbook.model ? ` · ${playbook.model}` : ''}
              </p>
              {playbook.network?.summary && (
                <p className="border border-steel/30 px-3 py-2 text-sm">
                  <span className="font-mono text-[11px] text-brass">{playbook.network.reading.toUpperCase()} </span>
                  {playbook.network.summary}
                </p>
              )}
              {playbook.circuit_classes && playbook.circuit_classes.length > 0 && (
                <p className="font-mono text-xs text-steel">
                  {playbook.circuit_classes.map((c) => `${c.code} ${c.class}`).join(' · ')}
                </p>
              )}
              {checks.some((c) => c.status !== 'open') && (
                <p className="border border-brass/40 bg-brass/10 px-3 py-2 text-sm">
                  <span className="font-mono text-[11px] tracking-widest text-brass">CHECKS </span>
                  {checks.filter((c) => c.status !== 'open').length} settled on this VIN. Rebuild to iterate from those findings.
                </p>
              )}
              {playbook.lookouts.length > 0 && (
                <div>
                  <p className="font-mono text-[11px] tracking-[0.2em] text-brass">LOOK OUT ON THIS CAR</p>
                  <ul className="mt-2 space-y-2">
                    {playbook.lookouts.map((l, i) => (
                      <li key={i} className="border border-brass/30 bg-brass/5 px-3 py-2">{l.text}</li>
                    ))}
                  </ul>
                </div>
              )}
              {playbook.likely_causes.length > 0 && (
                <div>
                  <p className="font-mono text-[11px] tracking-[0.2em] text-brass">LIKELY CAUSES</p>
                  <ul className="mt-2 space-y-2">
                    {playbook.likely_causes.map((c, i) => (
                      <li key={i} className="flex justify-between gap-3 border border-steel/20 px-3 py-2">
                        <span>{c.title}</span>
                        <span className="font-mono text-sm text-steel">{Math.round(c.probability * 100)}%</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {playbook.steps.length > 0 && (
                <ol className="space-y-3">
                  {playbook.steps.map((st) => {
                    const guides = guidesForStep(st)
                    const fp = checkFingerprint(st.kind, st.title)
                    const row = checks.find((c) => c.fingerprint === fp)
                    const status = row?.status ?? 'open'
                    const note = row?.note ?? ''
                    return (
                      <li key={st.order} className={`border-l-2 pl-3 ${status === 'done' ? 'border-ok' : status === 'ruled_out' ? 'border-fault' : 'border-brass'}`}>
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <p className="font-mono text-[11px] text-brass">
                            {st.order} · {st.kind.toUpperCase()}{st.adapter ? ' · ADAPTER' : ''}
                            {status === 'done' ? ' · DID THIS' : status === 'ruled_out' ? ' · NOT THIS' : ''}
                          </p>
                          <div className="flex flex-wrap items-center gap-1">
                            <IconBtn
                              tip={!online ? 'Ledger offline — reconnect to ask about this step' : 'Ask about this step using this scan, this shop, and retrieved pages'}
                              label="DETAILS"
                              disabled={!online}
                              onClick={() => setAskStep(st)}
                            >
                              <DetailIcon />
                            </IconBtn>
                            {guides.length > 0 && (
                              <IconBtn
                                tip="Beginner card: leads, dial, and what the display should look like — not pins on this car"
                                label="HOW-TO"
                                onClick={() => setHowTo(guides)}
                              >
                                <MeterIcon />
                              </IconBtn>
                            )}
                          </div>
                        </div>
                        <p className="font-semibold">{st.title}</p>
                        <p className="text-sm text-steel">{st.detail}</p>
                        {(st.pass || st.fail) && (
                          <p className="mt-1 text-sm">
                            {st.pass && <span className="text-ok">Pass: {st.pass} </span>}
                            {st.fail && <span className="text-fault">Fail: {st.fail}</span>}
                          </p>
                        )}
                        <label className="mt-2 block">
                          <span className="font-mono text-[11px] tracking-widest text-steel">FINDING</span>
                          <input
                            className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2 text-sm"
                            value={note}
                            placeholder="What you saw — no name / phone / plate"
                            onChange={(e) => {
                              const next = e.target.value
                              setChecks((rows) => {
                                const rest = rows.filter((c) => c.fingerprint !== fp)
                                return [...rest, {
                                  id: row?.id ?? fp,
                                  vin: (scan?.vin || vin).toUpperCase(),
                                  fingerprint: fp,
                                  kind: st.kind,
                                  title: st.title,
                                  detail: st.detail,
                                  status,
                                  note: next,
                                  updated_at: row?.updated_at ?? new Date().toISOString(),
                                }]
                              })
                            }}
                          />
                        </label>
                        <div className="mt-2 grid grid-cols-2 gap-2">
                          <button
                            type="button"
                            className={`min-h-11 border font-mono text-[11px] tracking-widest ${status === 'done' ? 'border-ok bg-ok/20 text-ok' : 'border-steel/30'}`}
                            onClick={() => persistCheck(st, 'done', note)}
                          >
                            DID THIS
                          </button>
                          <button
                            type="button"
                            className={`min-h-11 border font-mono text-[11px] tracking-widest ${status === 'ruled_out' ? 'border-fault bg-fault/20 text-fault' : 'border-steel/30'}`}
                            onClick={() => persistCheck(st, 'ruled_out', note)}
                          >
                            NOT THIS
                          </button>
                        </div>
                      </li>
                    )
                  })}
                </ol>
              )}
              {checks.filter((c) => {
                if (c.status === 'open') return false
                return !playbook.steps.some((st) => checkFingerprint(st.kind, st.title) === c.fingerprint)
              }).length > 0 && (
                <div>
                  <p className="font-mono text-[11px] tracking-[0.2em] text-brass">ALREADY ON THIS CAR</p>
                  <ul className="mt-2 space-y-1 text-sm text-steel">
                    {checks.filter((c) => c.status !== 'open' && !playbook.steps.some((st) => checkFingerprint(st.kind, st.title) === c.fingerprint)).map((c) => (
                      <li key={c.fingerprint} className="border border-steel/20 px-3 py-2">
                        <span className="font-mono text-[11px] text-brass">{c.status === 'done' ? 'DID THIS' : 'NOT THIS'} </span>
                        {c.title}{c.note ? ` — ${c.note}` : ''}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {playbook.validation && (
                <p className="text-sm"><span className="font-mono text-[11px] text-brass">VALIDATE </span>{playbook.validation}</p>
              )}
              {playbook.manual_figures && playbook.manual_figures.length > 0 && (
                <div>
                  <p className="font-mono text-[11px] tracking-[0.2em] text-brass">MANUAL PAGES</p>
                  <ul className="mt-2 space-y-1 text-sm text-steel">
                    {playbook.manual_figures.map((fig) => (
                      <li key={fig.id} className="border border-steel/20 px-3 py-2">
                        <p>{fig.kind ? fig.kind.toUpperCase() + ' · ' : ''}{fig.title} · p.{fig.page} · {fig.caption || fig.language}</p>
                        {fig.ocr_text && <p className="text-xs">On the picture: {fig.ocr_text}</p>}
                        {fig.image_url && <img alt={fig.caption || fig.title} className="mt-2 max-h-64 border border-steel/30" src={fig.image_url} />}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {playbook.gaps.length > 0 && (
                <ul className="space-y-1 text-sm text-steel">
                  {playbook.gaps.map((g, i) => <li key={i}>Gap: {g}</li>)}
                </ul>
              )}
            </div>
          )}
        </section>
      )}

      {page === 'job' && jobTab === 'close' && (
        <section className="mx-auto max-w-xl rounded-sm border border-brass/20 bg-panel p-5">
          <h2 className="mb-3 font-mono text-sm tracking-[0.25em] text-brass">CLOSEOUT</h2>
          <label className="mb-3 block">
            <span className="font-mono text-[11px] text-steel">MILEAGE KM</span>
            <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2 font-mono" value={mileage} onChange={(e) => setMileage(e.target.value)} placeholder="km" />
          </label>
          <div className="mb-4">
            <IconBtn
              tip={!scan ? 'Deep-scan first, or skip if you attached a report' : 'Persist this visit to the ledger'}
              label="LOG SESSION"
              disabled={!scan}
              onClick={() => run('log', async () => {
                if (!scan) return
                const body = {
                  vin: scan.vin || vin,
                  mileage_km: Number(mileage) || 0,
                  adapter_type: scan.adapter_type,
                  host_os: navigator.platform.toLowerCase().includes('win') ? 'windows' : 'linux',
                  protocol: scan.protocol,
                  active_codes: scan.active_codes,
                  freeze_frame: scan.freeze_frame,
                  raw_hex_stream: scan.raw_hex_stream,
                  captured_at: new Date().toISOString(),
                  make_hint: scan.make,
                  model_hint: scan.model,
                  year_hint: scan.year,
                }
                if (!online) {
                  enqueue({ kind: 'session', path: '/api/v1/sessions', body })
                  await persistCustomer()
                  setQueued(pendingCount())
                  return
                }
                const saved = await ingestSession(body)
                setSession(saved)
                await persistCustomer()
                applyHistory(await fetchHistory(saved.vin))
              }, 'Session logged', 'Close the job when you know what fixed it.')}
            >
              <StampIcon />
            </IconBtn>
          </div>
          <div className="mb-3 grid grid-cols-2 gap-2">
            <button type="button" className={`min-h-11 border ${outcome === 'success' ? 'border-ok bg-ok/20' : 'border-steel/30'}`} onClick={() => setOutcome('success')}>SUCCESS</button>
            <button type="button" className={`min-h-11 border ${outcome === 'failed' ? 'border-fault bg-fault/20' : 'border-steel/30'}`} onClick={() => setOutcome('failed')}>FAILED</button>
          </div>
          <textarea className="mb-3 min-h-24 w-full border border-steel/30 bg-oil px-3 py-2" placeholder="What actually fixed it?" value={rootCause} onChange={(e) => setRootCause(e.target.value)} />
          <input className="mb-3 w-full border border-steel/30 bg-oil px-3 py-2" value={parts} onChange={(e) => setParts(e.target.value)} placeholder="Parts replaced, comma-separated" />
          <IconBtn
            tip={!session ? 'Log the session (or attach a report) first' : !rootCause ? 'Write what actually fixed it' : 'Stamp this visit closed on this shop’s file'}
            label="CLOSE JOB"
            tone="paper"
            disabled={!session || !rootCause}
            onClick={() => run('closeout', async () => {
              if (!session) return
              const closeCodes = codes.length ? codes : (session.active_codes ?? [])
              const body = {
                outcome,
                diagnostic_trouble_code: closeCodes[0] ?? '',
                root_cause_explanation: rootCause,
                parts_replaced: parts.split(',').map((p) => p.trim()).filter(Boolean),
              }
              if (!online) {
                enqueue({ kind: 'closeout', path: `/api/v1/sessions/${session.id}/closeout`, body })
                await persistCustomer()
                setQueued(pendingCount())
                return
              }
              await closeoutSession(session.id, body)
              await persistCustomer()
              applyHistory(await fetchHistory(session.vin))
              goPage('file')
            }, 'Job closed', 'This shop’s file on this VIN now includes the work done.')}
          >
            <StampIcon />
          </IconBtn>
          <p className="mt-3 font-mono text-[11px] text-steel">Shop and technician are taken from your login, not the form.</p>
        </section>
      )}

      {howTo && <HowToModal guides={howTo} onClose={() => setHowTo(null)} />}
      {confirmClear && scan && (
        <ClearCodesModal
          codes={codes}
          modules={scan.modules}
          logged={Boolean(session)}
          busy={Boolean(busy)}
          onCancel={() => setConfirmClear(false)}
          onConfirm={() => {
            const remaining = codes
            void run(
              'clear codes',
              clearCodes,
              'Codes cleared',
              remaining.length
                ? 'Road test, then re-scan. They return if the fault is still there.'
                : 'No codes were stored on reachable modules.',
            )
          }}
        />
      )}
      {askStep && (
        <AskModal
          step={askStep}
          lookouts={playbook?.lookouts.map((l) => l.text) ?? []}
          payload={playbookPayload()}
          online={online}
          onClose={() => setAskStep(null)}
        />
      )}
      <ToastStack notices={notices} onDismiss={dismissNotice} />
    </div>
    {page === 'file' && canPrintFile && history && (
      <ShopFilePrint user={user} vin={vin} history={history} />
    )}
    </>
  )
}
