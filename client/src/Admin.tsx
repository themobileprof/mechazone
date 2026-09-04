/** Super admin: issue shops and technicians from landing tickets. */
import { useEffect, useMemo, useState } from 'react'
import { createShop, createTechnician, listAccessRequests, listShops, listTechnicians, logout, setAccessRequestStatus } from './api'
import { Logo } from './Brand'
import type { AccessRequest, Principal, Shop, Technician } from './types'

export function Admin({ user, onLogout }: { user: Principal; onLogout: () => void }) {
  const [shops, setShops] = useState<Shop[]>([])
  const [techs, setTechs] = useState<Technician[]>([])
  const [requests, setRequests] = useState<AccessRequest[]>([])
  const [error, setError] = useState<string | null>(null)
  const [shopName, setShopName] = useState('')
  const [shopCity, setShopCity] = useState('')
  const [shopCountry, setShopCountry] = useState('Nigeria')
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [shopId, setShopId] = useState('')

  const pending = useMemo(() => requests.filter((r) => r.status === 'pending'), [requests])
  const issuedCount = requests.filter((r) => r.status === 'provisioned').length
  const dismissedCount = requests.filter((r) => r.status === 'dismissed').length

  async function refresh() {
    const [s, t, a] = await Promise.all([listShops(), listTechnicians(), listAccessRequests()])
    setShops(s)
    setTechs(t)
    setRequests(a)
  }

  useEffect(() => {
    void refresh().catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
  }, [])

  function fillFromTicket(req: AccessRequest) {
    setFullName(req.applicant_name)
    setEmail(req.contact_email)
    setPassword('')
    if (req.kind === 'shop' && req.shop_name) {
      setShopName(req.shop_name)
      setShopCity(req.city)
      setShopCountry(req.country)
    } else {
      setShopId('')
    }
    document.getElementById('issue-login')?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  }

  return (
    <div className="relative flex min-h-svh flex-col px-5 py-4 md:h-svh md:overflow-hidden md:px-8">
      <div className="grain" />
      <header className="mb-4 flex shrink-0 flex-wrap items-center justify-between gap-4 border-b border-brass/30 pb-4">
        <div className="flex items-center gap-4">
          <Logo className="h-14 w-auto md:h-16" />
          <p className="font-mono text-[11px] tracking-[0.35em] text-brass">SUPER ADMIN</p>
        </div>
        <div className="flex items-center gap-4 font-mono text-xs">
          <span>{user.email}</span>
          <button className="border border-steel/40 px-3 py-2" onClick={() => void logout().then(onLogout)}>SIGN OUT</button>
        </div>
      </header>
      <p className="mb-4 max-w-2xl shrink-0 text-steel">You provision shops and technicians. Nobody self-registers. Pending tickets stay in the queue; issued people are under Logins.</p>
      {error && <p className="mb-4 shrink-0 border border-fault/40 bg-fault/10 px-3 py-2 text-fault">{error}</p>}

      <div className="grid min-h-0 flex-1 gap-4 md:grid-cols-3 md:overflow-hidden">
        <section className="flex min-h-0 flex-col border border-brass/20 bg-panel p-4">
          <h2 className="shrink-0 font-mono text-sm tracking-[0.25em] text-brass">QUEUE</h2>
          <p className="mt-1 shrink-0 text-sm text-steel">Pending requests only. Fill, then issue the login on the right.</p>
          <ol className="mt-3 min-h-0 flex-1 space-y-2 overflow-y-auto overscroll-contain pr-1 max-h-64 md:max-h-none">
            {pending.length === 0 && <li className="font-mono text-xs text-steel">No open tickets.</li>}
            {pending.map((req) => (
              <li key={req.id} className={`border px-3 py-3 ${email === req.contact_email ? 'border-brass/50' : 'border-steel/20'}`}>
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <p className="font-semibold">{req.applicant_name}</p>
                  <p className="font-mono text-[11px] tracking-[0.15em] text-brass">PENDING</p>
                </div>
                <p className="font-mono text-xs text-steel">
                  {req.contact_email}
                  {req.contact_phone ? ` · ${req.contact_phone}` : ''}
                  {' · '}
                  {req.kind === 'freelancer' ? 'freelancer' : req.shop_name}
                  {' · '}
                  {req.city}, {req.country}
                </p>
                {req.note && <p className="mt-1 text-sm text-steel">{req.note}</p>}
                <div className="mt-3 flex flex-wrap gap-2">
                  <button
                    className="border border-brass/40 px-3 py-2 font-mono text-[11px] text-brass"
                    type="button"
                    onClick={() => fillFromTicket(req)}
                  >
                    FILL FORMS
                  </button>
                  <button
                    className="border border-steel/40 px-3 py-2 font-mono text-[11px] text-steel"
                    type="button"
                    onClick={() => void setAccessRequestStatus(req.id, 'dismissed').then(refresh).catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))}
                  >
                    DISMISS
                  </button>
                </div>
              </li>
            ))}
          </ol>
          {(issuedCount > 0 || dismissedCount > 0) && (
            <p className="mt-3 shrink-0 border-t border-steel/20 pt-2 font-mono text-[11px] text-steel">
              {issuedCount > 0 ? `${issuedCount} issued — see Logins` : ''}
              {issuedCount > 0 && dismissedCount > 0 ? ' · ' : ''}
              {dismissedCount > 0 ? `${dismissedCount} dismissed` : ''}
            </p>
          )}
        </section>

        <section className="flex min-h-0 flex-col border border-brass/20 bg-panel p-4">
          <h2 className="shrink-0 font-mono text-sm tracking-[0.25em] text-brass">SHOPS</h2>
          <form className="mt-3 shrink-0 space-y-3" onSubmit={(e) => {
            e.preventDefault()
            void createShop({ name: shopName, location_city: shopCity, location_country: shopCountry })
              .then((shop) => { setShopName(''); setShopCity(''); setShopId(shop.id); return refresh() })
              .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
          }}>
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Shop name" value={shopName} onChange={(e) => setShopName(e.target.value)} required />
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="City" value={shopCity} onChange={(e) => setShopCity(e.target.value)} required />
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Country" value={shopCountry} onChange={(e) => setShopCountry(e.target.value)} />
            <button className="min-h-11 w-full bg-brass font-semibold text-oil" type="submit">CREATE SHOP</button>
          </form>
          <ol className="mt-4 min-h-0 flex-1 space-y-2 overflow-y-auto overscroll-contain pr-1 max-h-64 md:max-h-none">
            {shops.map((shop) => (
              <li key={shop.id} className="border border-steel/20 px-3 py-2">
                <p className="font-semibold">{shop.name}</p>
                <p className="font-mono text-xs text-steel">{shop.location_city}, {shop.location_country}</p>
              </li>
            ))}
          </ol>
        </section>

        <section id="issue-login" className="flex min-h-0 flex-col border border-brass/20 bg-panel p-4">
          <h2 className="shrink-0 font-mono text-sm tracking-[0.25em] text-brass">LOGINS</h2>
          <form className="mt-3 shrink-0 space-y-3" onSubmit={(e) => {
            e.preventDefault()
            void createTechnician({ full_name: fullName, email, password, shop_id: shopId })
              .then(() => { setFullName(''); setEmail(''); setPassword(''); return refresh() })
              .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
          }}>
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Full name" value={fullName} onChange={(e) => setFullName(e.target.value)} required />
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" type="email" placeholder="Login email" value={email} onChange={(e) => setEmail(e.target.value)} required />
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" type="password" placeholder="Temporary password (8+)" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
            <select className="w-full border border-steel/30 bg-oil px-3 py-2" value={shopId} onChange={(e) => setShopId(e.target.value)}>
              <option value="">Freelancer — no shop</option>
              {shops.map((shop) => (
                <option key={shop.id} value={shop.id}>{shop.name}</option>
              ))}
            </select>
            <button className="min-h-11 w-full bg-paper font-semibold text-oil" type="submit">ISSUE LOGIN</button>
          </form>
          <ol className="mt-4 min-h-0 flex-1 space-y-2 overflow-y-auto overscroll-contain pr-1 max-h-64 md:max-h-none">
            {techs.map((tech) => (
              <li key={tech.id} className="border border-steel/20 px-3 py-2">
                <p className="font-semibold">{tech.full_name}</p>
                <p className="font-mono text-xs text-steel">{tech.email} · {tech.freelancer ? 'freelancer' : tech.shop_name} · rep {tech.reputation_score}</p>
              </li>
            ))}
          </ol>
        </section>
      </div>
    </div>
  )
}
