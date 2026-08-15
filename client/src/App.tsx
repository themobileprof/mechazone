import { useEffect, useState } from 'react'
import { Admin } from './Admin'
import { me } from './api'
import { Bay } from './Bay'
import { Login } from './Login'
import type { Principal } from './types'

export default function App() {
  const [user, setUser] = useState<Principal | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    void me()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setReady(true))
  }, [])

  if (!ready) {
    return (
      <div className="flex min-h-svh items-center justify-center font-mono text-steel">
        LOADING…
      </div>
    )
  }
  if (!user) {
    return <Login onAuthed={setUser} />
  }
  if (user.role === 'super_admin') {
    return <Admin user={user} onLogout={() => setUser(null)} />
  }
  return <Bay user={user} onLogout={() => setUser(null)} />
}
