import { useState } from 'react'
import { login } from './api'
import type { Principal } from './types'

export function Login({ onAuthed }: { onAuthed: (p: Principal) => void }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  return (
    <div className="relative flex min-h-svh items-center justify-center px-5">
      <div className="grain" />
      <form
        className="w-full max-w-md border border-brass/30 bg-panel p-8"
        onSubmit={(e) => {
          e.preventDefault()
          setBusy(true)
          setError(null)
          void login(email, password)
            .then(onAuthed)
            .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
            .finally(() => setBusy(false))
        }}
      >
        <p className="font-mono text-[11px] tracking-[0.35em] text-brass">MECHAZONE</p>
        <h1 className="mt-2 text-3xl font-bold">Sign in</h1>
        <p className="mt-2 text-sm text-steel">Accounts are issued by a super admin. There is no self-signup.</p>
        <label className="mt-6 block">
          <span className="font-mono text-[11px] text-steel">EMAIL</span>
          <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-3" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        </label>
        <label className="mt-4 block">
          <span className="font-mono text-[11px] text-steel">PASSWORD</span>
          <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-3" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </label>
        {error && <p className="mt-4 border border-fault/40 bg-fault/10 px-3 py-2 text-sm text-fault">{error}</p>}
        <button className="mt-6 min-h-12 w-full bg-brass font-semibold text-oil" disabled={busy} type="submit">
          {busy ? 'CHECKING…' : 'ENTER BAY'}
        </button>
      </form>
    </div>
  )
}
