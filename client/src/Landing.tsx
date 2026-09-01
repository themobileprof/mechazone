import { useState, type FormEvent } from 'react'
import { requestAccess } from './api'
import { Login } from './Login'
import type { Principal } from './types'

type Kind = 'shop' | 'freelancer'

export function Landing({ onAuthed }: { onAuthed: (p: Principal) => void }) {
  const [signIn, setSignIn] = useState(false)
  const [kind, setKind] = useState<Kind>('shop')
  const [applicantName, setApplicantName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [shopName, setShopName] = useState('')
  const [city, setCity] = useState('')
  const [country, setCountry] = useState('')
  const [note, setNote] = useState('')
  const [website, setWebsite] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [queued, setQueued] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await requestAccess({
        applicant_name: applicantName,
        contact_email: email,
        contact_phone: phone,
        shop_name: shopName,
        city,
        country,
        kind,
        note,
        website,
      })
      setQueued(true)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="relative min-h-svh overflow-x-hidden">
      <div className="grain" />
      <div className="bay-wash" />

      <header className="relative z-10 mx-auto flex max-w-5xl items-center justify-between gap-4 px-5 py-5">
        <p className="text-lg font-bold tracking-wide text-brass">Mechazone</p>
        <button className="text-sm text-steel underline-offset-4 hover:text-paper hover:underline" type="button" onClick={() => setSignIn(true)}>
          I already have an account
        </button>
      </header>

      <section className="relative z-10 mx-auto grid max-w-5xl items-start gap-10 px-5 pb-16 pt-4 lg:grid-cols-2">
        <div>
          <p className="rise text-sm font-semibold text-brass">For independent workshops</p>
          <h1 className="rise d1 font-poster mt-3 text-[2.8rem] leading-[0.95] font-extrabold uppercase md:text-6xl">
            Scan the car. Find the fault. Finish the job.
          </h1>
          <p className="rise d2 mt-5 max-w-lg text-lg leading-relaxed text-paper">
            Mechazone puts a premium scanner, AI, and <strong className="text-paper">your workshop's job file</strong> in the bay — so you can diagnose modern cars, not guess from a cheap code reader.
          </p>
          <ol className="rise d3 mt-8 space-y-4 text-base">
            <li className="flex gap-3">
              <span className="font-bold text-brass">1.</span>
              <span>
                <strong className="text-paper">Premium scanner.</strong> OpenPort-class Pass-Thru. Read the real computers — engine, gearbox, and the modules a toy reader never sees.
              </span>
            </li>
            <li className="flex gap-3">
              <span className="font-bold text-brass">2.</span>
              <span>
                <strong className="text-paper">AI on this job.</strong> It uses the live scan plus the work this shop already did on this car — clear next steps, not generic internet advice.
              </span>
            </li>
            <li className="flex gap-3">
              <span className="font-bold text-brass">3.</span>
              <span>
                <strong className="text-paper">Your shop's job history.</strong> What you scanned, what you replaced, and what actually fixed it — on this car, in this workshop. It stays here; it is not a public vehicle record.
              </span>
            </li>
          </ol>
          <p className="mt-8 max-w-md text-sm text-steel">
            Today you request an account. Later you can pay for the kit and we register you. The laptop in the bay is enough — we do not send you a bargain dongle.
          </p>
        </div>

        <aside id="request" className="rise d2 border border-brass/40 bg-panel p-6">
          <h2 className="text-2xl font-bold">Request an account</h2>
          <p className="mt-2 text-sm text-steel">
            Fill this form. We will message you and open your login. You cannot create an account by yourself.
          </p>

          {queued ? (
            <div className="mt-6 border border-ok/40 bg-ok/10 px-4 py-5">
              <p className="text-lg font-semibold">We have your request.</p>
              <p className="mt-2 text-sm text-steel">
                We will contact <span className="text-paper">{phone}</span> to open your account.
              </p>
            </div>
          ) : (
            <form className="mt-5 space-y-3" onSubmit={(e) => void submit(e)}>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <KindBtn active={kind === 'shop'} onClick={() => setKind('shop')}>I have a workshop</KindBtn>
                <KindBtn active={kind === 'freelancer'} onClick={() => setKind('freelancer')}>I work alone</KindBtn>
              </div>
              <Field label="Your name" value={applicantName} onChange={setApplicantName} required />
              <Field label="Phone or WhatsApp" value={phone} onChange={setPhone} required placeholder="Include country code" />
              <Field label="Email (we send the login here)" type="email" value={email} onChange={setEmail} required />
              {kind === 'shop' && <Field label="Workshop name" value={shopName} onChange={setShopName} required />}
              <div className="grid grid-cols-2 gap-3">
                <Field label="City" value={city} onChange={setCity} required />
                <Field label="Country" value={country} onChange={setCountry} required />
              </div>
              <label className="block">
                <span className="text-sm text-steel">Cars you work on (optional)</span>
                <textarea
                  className="mt-1 min-h-16 w-full resize-y border border-steel/30 bg-oil px-3 py-2"
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  placeholder="Toyota, Honda, Chinese cars…"
                />
              </label>
              <label className="hp" aria-hidden="true">
                Website
                <input tabIndex={-1} autoComplete="off" value={website} onChange={(e) => setWebsite(e.target.value)} />
              </label>
              {error && <p className="border border-fault/40 bg-fault/10 px-3 py-2 text-sm text-fault">{error}</p>}
              <button className="min-h-12 w-full bg-brass text-base font-bold text-oil" disabled={busy} type="submit">
                {busy ? 'Sending…' : 'Send my request'}
              </button>
              <p className="text-xs text-steel">
                Your customers’ names and plate numbers stay on your laptop. We only keep the car’s mechanical record.
              </p>
            </form>
          )}
        </aside>
      </section>

      {signIn && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-oil/80 px-5">
          <button className="absolute inset-0 cursor-default" type="button" aria-label="Close" onClick={() => setSignIn(false)} />
          <div className="relative z-10 w-full max-w-md">
            <Login onAuthed={onAuthed} onCancel={() => setSignIn(false)} />
          </div>
        </div>
      )}
    </div>
  )
}

function KindBtn({ active, onClick, children }: { active: boolean; onClick: () => void; children: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`min-h-11 border px-2 text-left leading-tight ${active ? 'border-brass bg-brass text-oil' : 'border-steel/30 text-steel'}`}
    >
      {children}
    </button>
  )
}

function Field({
  label,
  value,
  onChange,
  type = 'text',
  required,
  placeholder,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  type?: string
  required?: boolean
  placeholder?: string
}) {
  return (
    <label className="block">
      <span className="text-sm text-steel">{label}</span>
      <input
        className="mt-1 w-full border border-steel/30 bg-oil px-3 py-2.5"
        type={type}
        value={value}
        required={required}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  )
}
