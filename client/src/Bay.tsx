import { useEffect, useMemo, useState } from 'react'
import { closeoutSession, decodeVin, fetchHistory, ingestSession, ledgerOnline, lookupDtc, logout } from './api'
import { enqueue, flushQueue, pendingCount } from './queue'
import type { HistoryResponse, Principal, ScanResult, Session } from './types'
import { worker } from './worker'

type Adapter = 'mock' | 'openport2_rev_e'

export function Bay({ user, onLogout }: { user: Principal; onLogout: () => void }) {
  const [online, setOnline] = useState(false)
  const [queued, setQueued] = useState(pendingCount())
  const [adapter, setAdapter] = useState<Adapter>('mock')
  const [connected, setConnected] = useState(false)
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [vin, setVin] = useState('')
  const [history, setHistory] = useState<HistoryResponse | null>(null)
  const [scan, setScan] = useState<ScanResult | null>(null)
  const [session, setSession] = useState<Session | null>(null)
  const [dtcTitles, setDtcTitles] = useState<Record<string, string>>({})
  const [mileage, setMileage] = useState('142500')
  const [localCustomer, setLocalCustomer] = useState('')
  const [outcome, setOutcome] = useState<'success' | 'failed'>('success')
  const [rootCause, setRootCause] = useState('')
  const [parts, setParts] = useState('Cleaned connector pins')

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

  const codes = scan?.active_codes ?? []

  useEffect(() => {
    void Promise.all(codes.map(async (code) => {
      try {
        const d = await lookupDtc(code)
        return [code, d.title || (d.cloud_ai_reserved ? 'Manufacturer-specific — use history/network' : '')] as const
      } catch {
        return [code, ''] as const
      }
    })).then((rows) => setDtcTitles(Object.fromEntries(rows)))
  }, [codes.join('|')])

  const vehicleLabel = useMemo(() => {
    if (history?.vehicle) {
      return `${history.vehicle.manufacture_year || ''} ${history.vehicle.make} ${history.vehicle.model}`.trim()
    }
    if (scan) return `${scan.year} ${scan.make} ${scan.model}`
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

      <section className="mb-6 grid gap-3 rounded-sm border border-brass/25 bg-panel/80 p-4 md:grid-cols-[1fr_auto_auto_auto] md:items-end">
        <label className="block">
          <span className="font-mono text-[11px] tracking-widest text-steel">ADAPTER</span>
          <select
            className="mt-1 w-full border border-steel/40 bg-oil px-3 py-3 text-paper"
            value={adapter}
            onChange={(e) => setAdapter(e.target.value as Adapter)}
          >
            <option value="mock">Mock ECU (bench)</option>
            <option value="openport2_rev_e">OpenPort 2.0 Rev E</option>
          </select>
        </label>
        <button className="min-h-12 border border-brass bg-brass px-5 font-semibold text-oil" onClick={() => run('connect', async () => {
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
          if (online) {
            await decodeVin(r.vin).catch(() => undefined)
            setHistory(await fetchHistory(r.vin))
          } else {
            setHistory({ vehicle: null, first_seen: true, sessions: [], resolutions: [] })
          }
        })}>
          READ VIN
        </button>
        <button className="min-h-12 border border-paper/40 px-5" disabled={!vin} onClick={() => run('scan', async () => {
          const result = await worker.scan()
          setScan(result)
          if (result.vin && result.vin !== vin) {
            setVin(result.vin)
            if (online) setHistory(await fetchHistory(result.vin))
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
              {history.first_seen ? 'FIRST SEEN ON NETWORK' : `${history.sessions.length} PRIOR SESSION(S)`}
            </span>
          )}
        </div>
        <label className="mt-4 block max-w-md">
          <span className="font-mono text-[11px] text-steel">CUSTOMER (LOCAL ONLY — NEVER SYNCED)</span>
          <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2" value={localCustomer} onChange={(e) => setLocalCustomer(e.target.value)} placeholder="Name / plate stay on this laptop" />
        </label>
      </section>

      <div className="grid gap-6 xl:grid-cols-[1.1fr_1fr_0.9fr]">
        <section className="rounded-sm border border-brass/20 bg-panel p-5">
          <h2 className="mb-3 font-mono text-sm tracking-[0.25em] text-brass">VIN TIMELINE</h2>
          {!history && <p className="text-steel">Read VIN first. History loads before a diagnosis.</p>}
          {history?.first_seen && <p className="text-steel">No ledger rows yet. This shop will open the book.</p>}
          <ol className="space-y-3">
            {history?.resolutions.map((res) => (
              <li key={res.id} className="border-l-2 border-ok pl-3">
                <p className="font-mono text-xs text-ok">{res.is_verified_fix ? 'VERIFIED FIX' : 'NOTE'} · {res.diagnostic_trouble_code}</p>
                <p>{res.root_cause_explanation}</p>
                <p className="text-sm text-steel">{res.parts_replaced.join(', ')}</p>
              </li>
            ))}
            {history?.sessions.map((sess) => (
              <li key={sess.id} className="border-l-2 border-steel/50 pl-3">
                <p className="font-mono text-xs text-steel">{new Date(sess.created_at).toLocaleString()} · {sess.outcome} · {sess.mileage_km} km</p>
                <p className="font-mono">{sess.active_codes.join('  ') || 'no codes'}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className="rounded-sm border border-brass/20 bg-panel p-5">
          <h2 className="mb-3 font-mono text-sm tracking-[0.25em] text-brass">LIVE MODULES</h2>
          {!scan && <p className="text-steel">Deep scan reads ECM / Valvematic over UDS. Not emissions PIDs.</p>}
          <div className="mb-4 flex flex-wrap gap-2">
            {scan?.modules.map((m) => (
              <span key={m.name} className={`border px-2 py-1 font-mono text-xs ${m.reachable ? 'border-ok text-ok' : 'border-fault text-fault'}`}>
                {m.name} {m.tx_id}
              </span>
            ))}
          </div>
          <ul className="mb-4 space-y-2">
            {codes.map((code) => (
              <li key={code} className="flex justify-between gap-3 border border-fault/30 bg-fault/5 px-3 py-2">
                <span className="font-mono text-fault">{code}</span>
                <span className="text-right text-sm text-steel">{dtcTitles[code]}</span>
              </li>
            ))}
          </ul>
          <dl className="grid grid-cols-2 gap-2 font-mono text-sm">
            {scan?.live.map((row) => (
              <div key={row.name} className="border border-steel/20 px-3 py-2">
                <dt className="text-[11px] text-steel">{row.name}</dt>
                <dd>{row.value} {row.unit}</dd>
              </div>
            ))}
          </dl>
        </section>

        <section className="rounded-sm border border-brass/20 bg-panel p-5">
          <h2 className="mb-3 font-mono text-sm tracking-[0.25em] text-brass">CLOSEOUT</h2>
          <label className="mb-3 block">
            <span className="font-mono text-[11px] text-steel">MILEAGE KM</span>
            <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2 font-mono" value={mileage} onChange={(e) => setMileage(e.target.value)} />
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
          <input className="mb-3 w-full border border-steel/30 bg-oil px-3 py-2" value={parts} onChange={(e) => setParts(e.target.value)} />
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
