import { useEffect, useMemo, useState } from 'react'
import { buildPlaybook, closeoutSession, decodeVin, fetchHistory, ingestSession, ledgerOnline, lookupDtc, logout } from './api'
import { enqueue, flushQueue, pendingCount } from './queue'
import type { DetectedAdapter, DidStream, HistoryResponse, Playbook, Principal, ScanCoverage, ScanResult, Session } from './types'
import { worker } from './worker'

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
  const [localCustomer, setLocalCustomer] = useState('')
  const [outcome, setOutcome] = useState<'success' | 'failed'>('success')
  const [rootCause, setRootCause] = useState('')
  const [parts, setParts] = useState('')
  const [playbook, setPlaybook] = useState<Playbook | null>(null)
  const [wiggle, setWiggle] = useState<DidStream | null>(null)
  const [dtcClass, setDtcClass] = useState<Record<string, string>>({})
  const [preCoverage, setPreCoverage] = useState<ScanCoverage | null>(null)

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

  const codes = scan?.active_codes ?? []
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

  const selectedKit = kits.find((k) => k.id === adapter)
  const coverage = scan?.coverage ?? preCoverage

  const vehicleLabel = useMemo(() => {
    if (history?.vehicle) {
      return `${history.vehicle.manufacture_year || ''} ${history.vehicle.make} ${history.vehicle.model}`.trim()
    }
    if (scan?.make || scan?.model) return `${scan.year || ''} ${scan.make} ${scan.model}`.trim()
    if (scan?.coverage?.depth === 'captured') return scan.profile.replaceAll('_', ' ')
    if (scan?.profile === 'generic_uds') return 'ISO 15765-4 probe — no captured OEM map'
    if (scan?.profile === 'toyota_common') return 'Toyota 11-bit probe'
    return 'Awaiting identify'
  }, [history, scan])

  async function run(label: string, fn: () => Promise<void>) {
    setBusy(label)
    setError(null)
    try {
      await fn()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(null)
    }
  }

  async function adviseFromScan(result: ScanResult, loggedVin: string) {
    const v = result.vin || loggedVin
    const book = await buildPlaybook({
      vin: v,
      make: history?.vehicle?.make || result.make,
      model: history?.vehicle?.model || result.model,
      year: history?.vehicle?.manufacture_year || result.year,
      engine_hint: result.profile,
      active_codes: result.active_codes ?? [],
      live: result.live,
      modules: result.modules,
      freeze_frame: result.freeze_frame,
      adapter_type: result.adapter_type,
      protocol: result.protocol,
      language: navigator.language.slice(0, 2) || 'en',
    })
    setPlaybook(book)
  }

  const affiliation = user.freelancer ? 'Freelancer' : (user.shop_name || 'Shop')

  return (
    <div className="relative min-h-svh px-5 py-4 md:px-8">
      <div className="grain" />
      <header className="mb-6 flex flex-wrap items-end justify-between gap-4 border-b border-brass/30 pb-4">
        <div>
          <p className="font-mono text-[11px] tracking-[0.35em] text-brass">BAY LEDGER</p>
          <h1 className="text-4xl font-bold tracking-wide text-paper md:text-5xl">MECHAZONE</h1>
        </div>
        <div className="flex flex-wrap items-center gap-3 font-mono text-xs">
          <span className={`h-2.5 w-2.5 rounded-full ${online ? 'bg-ok' : 'bg-fault'}`} />
          <span>{online ? 'LEDGER LIVE' : 'OFFLINE — QUEUEING'}</span>
          <span className="text-steel">Q {queued}</span>
          <span className="text-brass">{user.technician_name || user.email} · {affiliation}</span>
          <button className="border border-steel/40 px-3 py-2" onClick={() => void logout().then(onLogout)}>SIGN OUT</button>
        </div>
      </header>

      <section className="mb-6 grid gap-3 rounded-sm border border-brass/25 bg-panel/80 p-4 md:grid-cols-[1fr_auto_auto_auto_auto] md:items-end">
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
        <button className="min-h-12 border border-paper/40 px-4" onClick={() => run('detect', refreshKits)}>
          REFRESH KITS
        </button>
        <button className="min-h-12 border border-brass bg-brass px-5 font-semibold text-oil" disabled={selectedKit?.connectable === false} onClick={() => run('connect', async () => {
          const r = await worker.connectAdapter(adapter)
          setConnected(r.connected)
        })}>
          {connected ? 'RECONNECT' : 'CONNECT KIT'}
        </button>
        <button className="min-h-12 border border-paper/40 px-5" disabled={!connected} onClick={() => run('identify', async () => {
          const r = await worker.identify()
          setVin(r.vin)
          setScan(null)
          setSession(null)
          setPlaybook(null)
          setPreCoverage(r.coverage ?? null)
          if (online) {
            await decodeVin(r.vin).catch(() => undefined)
            setHistory(await fetchHistory(r.vin))
          } else {
            setHistory({ vehicle: null, first_seen: true, jobs: [], sessions: [], resolutions: [] })
          }
        })}>
          READ VIN
        </button>
        <button className="min-h-12 border border-paper/40 px-5" disabled={!vin} onClick={() => run(online ? 'scan + playbook' : 'scan', async () => {
          const result = await worker.scan({
            make: history?.vehicle?.make,
            model: history?.vehicle?.model,
            year: history?.vehicle?.manufacture_year,
          })
          setScan(result)
          setPlaybook(null)
          setWiggle(null)
          const v = result.vin || vin
          if (result.vin && result.vin !== vin) {
            setVin(result.vin)
          }
          if (online) {
            setHistory(await fetchHistory(v))
            await adviseFromScan(result, v)
          }
        })}>
          DEEP SCAN
        </button>
      </section>

      {error && <p className="mb-4 border border-fault/50 bg-fault/10 px-4 py-3 font-mono text-sm text-fault">{error}</p>}
      {busy && <p className="mb-4 font-mono text-sm text-brass">{busy.toUpperCase()}…</p>}

      <section className="mb-6 rounded-sm border border-brass/20 bg-panel p-5">
        <p className="font-mono text-[11px] tracking-[0.3em] text-steel">VEHICLE</p>
        <p className="mt-1 font-mono text-3xl tracking-[0.18em] md:text-5xl">{vin || '———— — ——————'}</p>
        <div className="mt-2 flex flex-wrap gap-4 text-sm text-steel">
          <span>{vehicleLabel}</span>
          {history && (
            <span className={history.first_seen ? 'text-brass' : 'text-ok'}>
              {history.first_seen ? 'FIRST VISIT TO THIS SHOP' : `${history.jobs?.length ?? history.sessions.length} JOB(S) HERE`}
            </span>
          )}
          {coverage && (
            <span className="font-mono text-xs text-brass">{coverage.id} · {coverage.depth.replaceAll('_', ' ')}</span>
          )}
        </div>
        {selectedKit?.gap && <p className="mt-2 text-sm text-steel">{selectedKit.gap}</p>}
        {coverage && coverage.gaps.length > 0 && (
          <div className="mt-3 border border-brass/30 bg-oil/60 px-3 py-3">
            <p className="font-mono text-[11px] tracking-widest text-brass">WHAT THIS VIN STILL NEEDS</p>
            <ul className="mt-2 space-y-1 text-sm text-steel">
              {coverage.gaps.map((g) => <li key={g}>{g}</li>)}
            </ul>
          </div>
        )}
        <label className="mt-4 block max-w-md">
          <span className="font-mono text-[11px] text-steel">CUSTOMER (LOCAL ONLY — NEVER SYNCED)</span>
          <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2" value={localCustomer} onChange={(e) => setLocalCustomer(e.target.value)} placeholder="Name / plate stay on this laptop" />
        </label>
      </section>

      <section className="mb-6 rounded-sm border border-brass/20 bg-panel p-5">
        <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="font-mono text-sm tracking-[0.25em] text-brass">AI PLAYBOOK</h2>
            <p className="mt-1 text-sm text-steel">After every deep scan it uses the faults, live data, and this shop's jobs on this car — lookouts and next tests, not a code dump. Work stays in this shop.</p>
          </div>
          <button
            className="min-h-12 bg-brass px-5 font-semibold text-oil"
            disabled={!scan || !vin || !online}
            onClick={() => run('playbook', async () => {
              if (!scan) return
              await adviseFromScan(scan, vin)
            })}
          >
            REBUILD PLAYBOOK
          </button>
        </div>
        {!playbook && <p className="text-steel">Deep scan writes the playbook when the ledger is online. Offline, scan first and rebuild when you reconnect.</p>}
        {playbook && (
          <div className="space-y-4">
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
                {playbook.steps.map((st) => (
                  <li key={st.order} className="border-l-2 border-brass pl-3">
                    <p className="font-mono text-[11px] text-brass">{st.order} · {st.kind.toUpperCase()}{st.adapter ? ' · ADAPTER' : ''}</p>
                    <p className="font-semibold">{st.title}</p>
                    <p className="text-sm text-steel">{st.detail}</p>
                    {(st.pass || st.fail) && (
                      <p className="mt-1 text-sm">
                        {st.pass && <span className="text-ok">Pass: {st.pass} </span>}
                        {st.fail && <span className="text-fault">Fail: {st.fail}</span>}
                      </p>
                    )}
                  </li>
                ))}
              </ol>
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

      <div className="grid gap-6 xl:grid-cols-[1.1fr_1fr_0.9fr]">
        <section className="rounded-sm border border-brass/20 bg-panel p-5">
          <h2 className="mb-3 font-mono text-sm tracking-[0.25em] text-brass">THIS SHOP'S WORK</h2>
          {!history && <p className="text-steel">Read VIN first. This shop's jobs load before a diagnosis.</p>}
          {history?.first_seen && <p className="text-steel">This shop has not worked this vehicle yet. Close the job to start the file.</p>}
          <ol className="space-y-3">
            {(history?.jobs ?? []).map((job) => (
              <li key={job.session_id} className={`border-l-2 pl-3 ${job.verified_fix ? 'border-ok' : 'border-steel/50'}`}>
                <p className="font-mono text-xs text-steel">
                  {new Date(job.created_at).toLocaleString()} · {job.technician_name || 'tech'} · {job.mileage_km} km · {job.outcome}
                  {job.verified_fix ? ' · CLOSED' : ''}
                </p>
                {job.work ? (
                  <p>{job.work}</p>
                ) : (
                  <p className="text-sm text-steel">Scan logged — close the job to record the work done.</p>
                )}
                {job.parts_replaced.length > 0 && (
                  <p className="text-sm text-steel">Parts: {job.parts_replaced.join(', ')}</p>
                )}
                {(job.closeout_code || job.active_codes.length > 0) && (
                  <p className="font-mono text-xs text-steel">
                    {job.closeout_code || job.active_codes.join('  ')}
                  </p>
                )}
              </li>
            ))}
          </ol>
        </section>

        <section className="rounded-sm border border-brass/20 bg-panel p-5">
          <h2 className="mb-3 font-mono text-sm tracking-[0.25em] text-brass">LIVE MODULES</h2>
          {!scan && <p className="text-steel">Deep scan uses the VIN profile: a captured platform map when we have one, otherwise a Toyota 11-bit probe or ISO 15765-4 (7E0–7E2). Dark means no UDS answer — not a generic PID miss.</p>}
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
          <button
            className="mt-4 min-h-11 w-full border border-brass/50 px-3 text-sm text-brass"
            disabled={!connected || !scan || !(scan.live?.length)}
            onClick={() => run('wiggle', async () => {
              setWiggle(await worker.streamDids(6))
            })}
          >
            WIGGLE LOG (ECM DIDS)
          </button>
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

        <section className="rounded-sm border border-brass/20 bg-panel p-5">
          <h2 className="mb-3 font-mono text-sm tracking-[0.25em] text-brass">CLOSEOUT</h2>
          <label className="mb-3 block">
            <span className="font-mono text-[11px] text-steel">MILEAGE KM</span>
            <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2 font-mono" value={mileage} onChange={(e) => setMileage(e.target.value)} placeholder="km" />
          </label>
          <button className="mb-4 min-h-12 w-full border border-brass px-4 text-brass" disabled={!scan} onClick={() => run('log', async () => {
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
              setQueued(pendingCount())
              return
            }
            const saved = await ingestSession(body)
            setSession(saved)
            setHistory(await fetchHistory(saved.vin))
          })}>
            LOG SESSION TO LEDGER
          </button>
          <div className="mb-3 grid grid-cols-2 gap-2">
            <button className={`min-h-11 border ${outcome === 'success' ? 'border-ok bg-ok/20' : 'border-steel/30'}`} onClick={() => setOutcome('success')}>SUCCESS</button>
            <button className={`min-h-11 border ${outcome === 'failed' ? 'border-fault bg-fault/20' : 'border-steel/30'}`} onClick={() => setOutcome('failed')}>FAILED</button>
          </div>
          <textarea className="mb-3 min-h-24 w-full border border-steel/30 bg-oil px-3 py-2" placeholder="What actually fixed it?" value={rootCause} onChange={(e) => setRootCause(e.target.value)} />
          <input className="mb-3 w-full border border-steel/30 bg-oil px-3 py-2" value={parts} onChange={(e) => setParts(e.target.value)} placeholder="Parts replaced, comma-separated" />
          <button className="min-h-12 w-full bg-paper font-semibold text-oil" disabled={!session || !rootCause} onClick={() => run('closeout', async () => {
            if (!session) return
            const body = {
              outcome,
              diagnostic_trouble_code: codes[0] ?? '',
              root_cause_explanation: rootCause,
              parts_replaced: parts.split(',').map((p) => p.trim()).filter(Boolean),
            }
            if (!online) {
              enqueue({ kind: 'closeout', path: `/api/v1/sessions/${session.id}/closeout`, body })
              setQueued(pendingCount())
              return
            }
            await closeoutSession(session.id, body)
            setHistory(await fetchHistory(session.vin))
          })}>
            CLOSE JOB
          </button>
          <p className="mt-3 font-mono text-[11px] text-steel">Shop and technician are taken from your login, not the form.</p>
        </section>
      </div>
    </div>
  )
}
