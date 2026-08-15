import { useState } from 'react'
import { login } from './api'
import type { Principal } from './types'

export function Login({
  onAuthed,
  onCancel,
}: {
  onAuthed: (p: Principal) => void
  onCancel?: () => void
}) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  return (
    <form
      className="w-full border border-brass/30 bg-panel p-8"
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
      <h1 className="text-3xl font-bold">Sign in</h1>
      <p className="mt-2 text-sm text-steel">Use the email and password we sent you after your request.</p>
      <label className="mt-6 block">
        <span className="text-sm text-steel">Email</span>
        <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-3" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
      </label>
      <label className="mt-4 block">
        <span className="text-sm text-steel">Password</span>
        <input className="mt-1 w-full border border-steel/30 bg-oil px-3 py-3" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
      </label>
      {error && <p className="mt-4 border border-fault/40 bg-fault/10 px-3 py-2 text-sm text-fault">{error}</p>}
      <button className="mt-6 min-h-12 w-full bg-brass font-semibold text-oil" disabled={busy} type="submit">
        {busy ? 'Checking…' : 'Sign in'}
      </button>
      {onCancel && (
        <button className="mt-3 min-h-10 w-full text-sm text-steel" type="button" onClick={onCancel}>
          Back
        </button>
      )}
    </form>
  )
}
