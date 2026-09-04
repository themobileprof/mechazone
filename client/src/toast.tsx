/** Viewport-fixed job notices. Success fades; faults stay until dismissed. */
import { useEffect } from 'react'

export type Notice = {
  id: number
  kind: 'ok' | 'fault' | 'busy'
  title: string
  detail?: string
}

export function ToastStack({ notices, onDismiss }: { notices: Notice[]; onDismiss: (id: number) => void }) {
  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex justify-center p-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
      <ul className="flex w-full max-w-xl flex-col gap-2">
        {notices.map((n) => (
          <li
            key={n.id}
            className={`notice-card pointer-events-auto border px-4 py-3 shadow-[0_12px_40px_rgba(0,0,0,0.45)] ${
              n.kind === 'fault'
                ? 'border-fault bg-oil text-fault'
                : n.kind === 'busy'
                  ? 'border-brass/60 bg-panel text-brass'
                  : 'border-ok/70 bg-oil text-ok'
            }`}
          >
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="font-mono text-[11px] tracking-[0.28em]">{n.kind === 'fault' ? 'FAILED' : n.kind === 'busy' ? 'WORKING' : 'DONE'}</p>
                <p className="mt-1 font-semibold text-paper">{n.title}</p>
                {n.detail && <p className="mt-1 text-sm text-steel">{n.detail}</p>}
              </div>
              {n.kind !== 'busy' && (
                <button className="font-mono text-xs text-steel" onClick={() => onDismiss(n.id)} type="button">
                  DISMISS
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

export function useAutoDismiss(notices: Notice[], onDismiss: (id: number) => void, ms = 4200) {
  useEffect(() => {
    const timers = notices.filter((n) => n.kind === 'ok').map((n) => window.setTimeout(() => onDismiss(n.id), ms))
    return () => timers.forEach((t) => window.clearTimeout(t))
  }, [notices, onDismiss, ms])
}
