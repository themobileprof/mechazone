import { useEffect, useId, useRef, useState } from 'react'
import { ClearCodesIcon, IconBtn } from './chrome'
import type { ScanModule } from './types'

export function ClearCodesModal({
  codes,
  modules,
  logged,
  busy,
  onCancel,
  onConfirm,
}: {
  codes: string[]
  modules: ScanModule[]
  logged: boolean
  busy: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  const titleId = useId()
  const closeRef = useRef<HTMLButtonElement>(null)
  const [ack, setAck] = useState(false)
  const targets = modules.filter((m) => m.reachable && (m.dtcs?.length ?? 0) > 0)

  useEffect(() => {
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onCancel])

  return (
    <div className="howto-overlay" onClick={onCancel} role="presentation">
      <div
        className="howto-sheet"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-brass/25 pb-3">
          <div>
            <p className="font-mono text-[11px] tracking-[0.3em] text-brass">UDS $14 · ALL GROUPS</p>
            <h2 id={titleId} className="mt-1 font-poster text-2xl tracking-wide text-paper">
              CLEAR CODES
            </h2>
            <p className="mt-1 max-w-xl text-sm text-steel">
              This does not repair the car. Codes and freeze-frame on those modules go. They come back if the fault is still there.
            </p>
          </div>
          <button
            ref={closeRef}
            type="button"
            className="min-h-11 border border-paper/40 px-4 font-mono text-[11px] tracking-widest text-paper hover:border-brass hover:text-brass"
            onClick={onCancel}
          >
            BACK
          </button>
        </div>
        <ul className="mt-4 list-disc space-y-2 pl-5 text-sm text-steel">
          <li>Ignition on, same as a scan. Not a programming write, not $2F IO-control.</li>
          <li>Only modules that answered and currently have codes. Dark nodes are left alone.</li>
          <li>If an ECU NRC-rejects $14, we stop on that node — no seed or key.</li>
          <li>
            {logged
              ? 'This visit is already on the job file. Clearing updates the live bus map, not that log.'
              : 'Log the session on Close if this shop should keep these codes on the job file. The last deep-scan capture already has them until this clear.'}
          </li>
        </ul>
        {codes.length > 0 && (
          <p className="mt-4 font-mono text-sm text-fault">{codes.join('  ')}</p>
        )}
        {targets.length > 0 && (
          <p className="mt-2 font-mono text-xs text-steel">
            {targets.map((m) => `${m.name} ${m.tx_id}`).join(' · ')}
          </p>
        )}
        <label className="mt-4 flex items-start gap-3 text-sm text-paper">
          <input
            type="checkbox"
            className="mt-1"
            checked={ack}
            onChange={(e) => setAck(e.target.checked)}
          />
          <span>I understand this is not a repair. Road test, then re-scan.</span>
        </label>
        <div className="mt-5">
          <IconBtn
            tip={!ack ? 'Tick the acknowledgement first' : busy ? 'Sending $14' : 'Send UDS $14 on the modules above'}
            label="CLEAR CODES"
            tone="brass"
            disabled={!ack || busy}
            onClick={onConfirm}
          >
            <ClearCodesIcon />
          </IconBtn>
        </div>
      </div>
    </div>
  )
}
