import { useEffect, useState } from 'react'
import { createShop, createTechnician, listShops, listTechnicians, logout } from './api'
import type { Principal, Shop, Technician } from './types'

export function Admin({ user, onLogout }: { user: Principal; onLogout: () => void }) {
  const [shops, setShops] = useState<Shop[]>([])
  const [techs, setTechs] = useState<Technician[]>([])
  const [error, setError] = useState<string | null>(null)
  const [shopName, setShopName] = useState('')
  const [shopCity, setShopCity] = useState('')
  const [shopCountry, setShopCountry] = useState('Nigeria')
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [shopId, setShopId] = useState('')

  async function refresh() {
    const [s, t] = await Promise.all([listShops(), listTechnicians()])
    setShops(s)
    setTechs(t)
  }

  useEffect(() => {
    void refresh().catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
  }, [])

  return (
    <div className="relative min-h-svh px-5 py-4 md:px-8">
      <div className="grain" />
      <header className="mb-6 flex flex-wrap items-end justify-between gap-4 border-b border-brass/30 pb-4">
        <div>
          <p className="font-mono text-[11px] tracking-[0.35em] text-brass">SUPER ADMIN</p>
          <h1 className="text-4xl font-bold">MECHAZONE</h1>
        </div>
        <div className="flex items-center gap-4 font-mono text-xs">
          <span>{user.email}</span>
          <button className="border border-steel/40 px-3 py-2" onClick={() => void logout().then(onLogout)}>SIGN OUT</button>
        </div>
      </header>
      <p className="mb-6 max-w-2xl text-steel">You provision shops and technicians (including freelancers). They sign in with the email and password you set. Nobody self-registers.</p>
      {error && <p className="mb-4 border border-fault/40 bg-fault/10 px-3 py-2 text-fault">{error}</p>}

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="border border-brass/20 bg-panel p-5">
          <h2 className="font-mono text-sm tracking-[0.25em] text-brass">ADD SHOP</h2>
          <form className="mt-4 space-y-3" onSubmit={(e) => {
            e.preventDefault()
            void createShop({ name: shopName, location_city: shopCity, location_country: shopCountry })
              .then(() => { setShopName(''); setShopCity(''); return refresh() })
              .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
          }}>
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Shop name" value={shopName} onChange={(e) => setShopName(e.target.value)} required />
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="City" value={shopCity} onChange={(e) => setShopCity(e.target.value)} required />
            <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Country" value={shopCountry} onChange={(e) => setShopCountry(e.target.value)} />
            <button className="min-h-11 w-full bg-brass font-semibold text-oil" type="submit">CREATE SHOP</button>
          </form>
          <ol className="mt-5 space-y-2">
            {shops.map((shop) => (
              <li key={shop.id} className="border border-steel/20 px-3 py-2">
                <p className="font-semibold">{shop.name}</p>
                <p className="font-mono text-xs text-steel">{shop.location_city}, {shop.location_country}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className="border border-brass/20 bg-panel p-5">
          <h2 className="font-mono text-sm tracking-[0.25em] text-brass">ADD TECHNICIAN</h2>
          <form className="mt-4 space-y-3" onSubmit={(e) => {
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
          <ol className="mt-5 space-y-2">
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
