/** Super admin: issue shops and technicians from landing tickets. */
import { useEffect, useMemo, useState } from 'react'
import { createShop, createTechnician, listAccessRequests, listShops, listTechnicians, logout, setAccessRequestStatus } from './api'
import { AdminHowTo } from './AdminHowTo'
import { Logo } from './Brand'
import { BookIcon, DismissIcon, IconBtn, LoginIcon, QueueIcon, ShopIcon, SignOutIcon, Tip } from './chrome'
import type { AccessRequest, Principal, Shop, Technician } from './types'

type Desk = 'queue' | 'shops' | 'logins' | 'howto'

const DESKS: { id: Desk; n: string; label: string; hint: string }[] = [
  { id: 'queue', n: '01', label: 'QUEUE', hint: 'Pending tickets — fill, then provision' },
  { id: 'shops', n: '02', label: 'SHOPS', hint: 'Create the workshop if it is new' },
  { id: 'logins', n: '03', label: 'LOGINS', hint: 'Issue the technician account' },
  { id: 'howto', n: '04', label: 'HOW-TO', hint: 'Bay cards for morphed AI actions' },
]

export function Admin({ user, onLogout }: { user: Principal; onLogout: () => void }) {
  const [desk, setDesk] = useState<Desk>('queue')
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
      setDesk('shops')
    } else {
      setShopId('')
      setDesk('logins')
    }
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
          <IconBtn tip="Sign out" onClick={() => void logout().then(onLogout)}>
            <SignOutIcon />
          </IconBtn>
        </div>
      </header>
      <p className="mb-4 max-w-2xl shrink-0 text-steel">You provision shops and technicians, and you write the how-to cards the bay shows on playbook steps. Nobody self-registers.</p>
      {error && <p className="mb-4 shrink-0 border border-fault/40 bg-fault/10 px-3 py-2 text-fault">{error}</p>}

      <nav className="desk-tabs shrink-0" aria-label="Admin desks">
        {DESKS.map((d) => (
          <Tip key={d.id} label={d.hint}>
            <button
              type="button"
              className={`desk-tab ${desk === d.id ? 'is-active' : ''}`}
              aria-current={desk === d.id ? 'page' : undefined}
              onClick={() => setDesk(d.id)}
            >
              {d.id === 'queue' ? <QueueIcon /> : d.id === 'shops' ? <ShopIcon /> : d.id === 'howto' ? <BookIcon /> : <LoginIcon />}
              <span>{d.n} {d.label}</span>
              {d.id === 'queue' && pending.length > 0 && (
                <span className="font-mono text-[10px] text-brass">{pending.length}</span>
              )}
            </button>
          </Tip>
        ))}
      </nav>
      <p className="mb-3 shrink-0 font-mono text-[11px] tracking-wide text-steel">{DESKS.find((d) => d.id === desk)?.hint}</p>

      <div className="min-h-0 flex-1 md:overflow-hidden">
        {desk === 'queue' && (
        <section className="flex h-full min-h-0 flex-col border border-brass/20 bg-panel p-4">
          <h2 className="shrink-0 font-mono text-sm tracking-[0.25em] text-brass">QUEUE</h2>
          <p className="mt-1 shrink-0 text-sm text-steel">Pending requests only. Fill, then create the shop if needed and issue the login.</p>
          <ol className="mt-3 min-h-0 flex-1 space-y-2 overflow-y-auto overscroll-contain pr-1">
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
                  <IconBtn
                    tip={req.kind === 'shop' ? 'Copy into Shops, then issue the login' : 'Copy into Logins'}
                    label="FILL"
                    tone="brass"
                    onClick={() => fillFromTicket(req)}
                  >
                    <QueueIcon />
                  </IconBtn>
                  <IconBtn
                    tip="Drop this ticket without issuing a login"
                    label="DISMISS"
                    tone="fault"
                    onClick={() => void setAccessRequestStatus(req.id, 'dismissed').then(refresh).catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))}
                  >
                    <DismissIcon />
                  </IconBtn>
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
        )}

        {desk === 'shops' && (
        <section className="flex h-full min-h-0 flex-col border border-brass/20 bg-panel p-4">
          <h2 className="shrink-0 font-mono text-sm tracking-[0.25em] text-brass">SHOPS</h2>
          <form className="mt-3 shrink-0 space-y-3" onSubmit={(e) => {
            e.preventDefault()
            void createShop({ name: shopName, location_city: shopCity, location_country: shopCountry })
              .then((shop) => { setShopName(''); setShopCity(''); setShopId(shop.id); setDesk('logins'); return refresh() })
              .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
          }}>
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Shop name" value={shopName} onChange={(e) => setShopName(e.target.value)} required />
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="City" value={shopCity} onChange={(e) => setShopCity(e.target.value)} required />
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Country" value={shopCountry} onChange={(e) => setShopCountry(e.target.value)} />
            <IconBtn tip="Create this workshop, then issue its first login" label="CREATE SHOP" tone="brass" type="submit">
              <ShopIcon />
            </IconBtn>
          </form>
          <ol className="mt-4 min-h-0 flex-1 space-y-2 overflow-y-auto overscroll-contain pr-1">
            {shops.map((shop) => (
              <li key={shop.id} className="border border-steel/20 px-3 py-2">
                <p className="font-semibold">{shop.name}</p>
                <p className="font-mono text-xs text-steel">{shop.location_city}, {shop.location_country}</p>
              </li>
            ))}
          </ol>
        </section>
        )}

        {desk === 'logins' && (
        <section className="flex h-full min-h-0 flex-col border border-brass/20 bg-panel p-4">
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
            <IconBtn tip="Provision this technician — they cannot self-register" label="ISSUE LOGIN" tone="paper" type="submit">
              <LoginIcon />
            </IconBtn>
          </form>
          <ol className="mt-4 min-h-0 flex-1 space-y-2 overflow-y-auto overscroll-contain pr-1">
            {techs.map((tech) => (
              <li key={tech.id} className="border border-steel/20 px-3 py-2">
                <p className="font-semibold">{tech.full_name}</p>
                <p className="font-mono text-xs text-steel">{tech.email} · {tech.freelancer ? 'freelancer' : tech.shop_name} · rep {tech.reputation_score}</p>
              </li>
            ))}
          </ol>
        </section>
        )}

        {desk === 'howto' && <AdminHowTo onError={setError} />}
      </div>
    </div>
  )
}
