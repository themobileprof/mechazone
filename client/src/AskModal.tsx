import { useEffect, useId, useRef, useState } from 'react'
import { askPlaybook, type PlaybookBody } from './api'
import { DetailIcon, IconBtn } from './chrome'
import type { PlaybookAsk, PlaybookStep } from './types'

const STARTERS = [
  'Walk me through this test on this car.',
  'What should I see if this passes?',
  'Which retrieved page or figure covers this?',
  'If this fails, what is the next check from this scan?',
]

type Turn = { role: 'user' | 'assistant'; content: string; gaps?: string[]; figures?: PlaybookAsk['figures'] }

export function AskModal({
  step,
  lookouts,
  payload,
  online,
  onClose,
}: {
  step: PlaybookStep
  lookouts: string[]
  payload: PlaybookBody | null
  online: boolean
  onClose: () => void
}) {
  const titleId = useId()
  const closeRef = useRef<HTMLButtonElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const threadEnd = useRef<HTMLDivElement>(null)
  const [draft, setDraft] = useState('')
  const [turns, setTurns] = useState<Turn[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  useEffect(() => {
    threadEnd.current?.scrollIntoView({ block: 'nearest' })
  }, [turns, busy])

  async function send(text: string) {
    const question = text.trim()
    if (!question || busy) return
    if (!payload) {
      setError('Deep-scan or attach a report first.')
      return
    }
    if (!online) {
      setError('Ledger offline — reconnect to ask.')
      return
    }
    setError(null)
    setDraft('')
    const nextTurns = [...turns, { role: 'user' as const, content: question }]
    setTurns(nextTurns)
    setBusy(true)
    try {
      const reply = await askPlaybook({
        ...payload,
        step,
        question,
        lookouts,
        thread: nextTurns.slice(0, -1).map((t) => ({ role: t.role, content: t.content })),
      })
      setTurns((rows) => [
        ...rows,
        {
          role: 'assistant',
          content: reply.answer,
          gaps: reply.gaps,
          figures: reply.figures,
        },
      ])
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
      setTurns((rows) => rows.slice(0, -1))
      setDraft(question)
    } finally {
      setBusy(false)
      inputRef.current?.focus()
    }
  }

  return (
    <div className="howto-overlay" onClick={onClose} role="presentation">
      <div
        className="howto-sheet ask-sheet"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-brass/25 pb-3">
          <div>
            <p className="font-mono text-[11px] tracking-[0.3em] text-brass">DETAILS · THIS STEP</p>
            <h2 id={titleId} className="mt-1 font-poster text-2xl tracking-wide text-paper">
              {step.title}
            </h2>
            <p className="mt-1 font-mono text-[11px] text-steel">
              {String(step.order).padStart(2, '0')} · {step.kind.toUpperCase()}
              {step.adapter ? ' · ADAPTER' : ''}
            </p>
          </div>
          <button
            ref={closeRef}
            type="button"
            className="min-h-11 border border-paper/40 px-4 font-mono text-[11px] tracking-widest text-paper hover:border-brass hover:text-brass"
            onClick={onClose}
          >
            CLOSE
          </button>
        </div>

        <p className="mt-3 border-l-2 border-brass/50 pl-3 text-sm text-steel">{step.detail}</p>
        {(step.pass || step.fail) && (
          <p className="mt-2 text-sm">
            {step.pass && <span className="text-ok">Pass: {step.pass} </span>}
            {step.fail && <span className="text-fault">Fail: {step.fail}</span>}
          </p>
        )}
        <p className="mt-3 border border-brass/40 bg-brass/10 px-3 py-2 text-sm">
          Answers use this scan, this shop’s jobs, and retrieved pages. Uncited pins and drawings stay in gaps.
        </p>

        <ol className="ask-thread mt-4 space-y-3">
          {turns.map((t, i) => (
            <li key={`${t.role}-${i}`} className={`ask-turn ask-turn-${t.role}`}>
              <p className="font-mono text-[11px] tracking-[0.25em] text-brass">
                {String(i + 1).padStart(2, '0')} · {t.role === 'user' ? 'YOU' : 'SHOP BOOK'}
              </p>
              <p className="mt-1 whitespace-pre-wrap text-sm">{t.content || 'No answer.'}</p>
              {t.figures && t.figures.length > 0 && (
                <ul className="mt-2 space-y-2">
                  {t.figures.map((fig) => (
                    <li key={fig.id} className="border border-steel/20 px-3 py-2 text-sm text-steel">
                      <p>{fig.kind ? fig.kind.toUpperCase() + ' · ' : ''}{fig.title} · p.{fig.page} · {fig.caption || fig.language}</p>
                      {fig.image_url && <img alt={fig.caption || fig.title} className="mt-2 max-h-48 border border-steel/30" src={fig.image_url} />}
                    </li>
                  ))}
                </ul>
              )}
              {t.gaps && t.gaps.length > 0 && (
                <ul className="mt-2 space-y-1 text-sm text-steel">
                  {t.gaps.map((g) => <li key={g}>Gap: {g}</li>)}
                </ul>
              )}
            </li>
          ))}
          {busy && (
            <li className="ask-turn ask-turn-assistant">
              <p className="font-mono text-[11px] tracking-[0.25em] text-brass">ASKING THE BOOK…</p>
              <p className="mt-1 text-sm text-steel">Retrieving this shop’s jobs and matching pages for this step.</p>
            </li>
          )}
        </ol>
        <div ref={threadEnd} />

        {turns.length === 0 && !busy && (
          <div className="mt-4 flex flex-wrap gap-1">
            {STARTERS.map((line) => (
              <button
                key={line}
                type="button"
                className="min-h-10 border border-steel/30 px-3 text-left font-mono text-[11px] tracking-wide text-steel hover:border-brass hover:text-brass"
                onClick={() => setDraft(line)}
              >
                {line}
              </button>
            ))}
          </div>
        )}

        {error && <p className="mt-3 border border-fault/50 bg-fault/10 px-3 py-2 text-sm text-fault">{error}</p>}

        <form
          className="mt-4 border-t border-brass/20 pt-3"
          onSubmit={(e) => {
            e.preventDefault()
            void send(draft)
          }}
        >
          <label className="block">
            <span className="font-mono text-[11px] tracking-widest text-steel">ASK ABOUT THIS STEP</span>
            <textarea
              ref={inputRef}
              className="mt-1 min-h-24 w-full resize-y border border-steel/30 bg-oil px-3 py-2 text-sm"
              value={draft}
              maxLength={2000}
              placeholder="No name, phone, or plate. Ask how to run this test on this car."
              disabled={busy}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                  e.preventDefault()
                  void send(draft)
                }
              }}
            />
          </label>
          <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
            <p className="font-mono text-[11px] text-steel">Ctrl+Enter to send</p>
            <IconBtn
              tip={!online ? 'Ledger offline — reconnect to ask' : !payload ? 'Capture a scan or report first' : 'Ask using this step, this scan, and retrieved pages'}
              label={busy ? 'ASKING' : 'ASK'}
              tone="brass"
              type="submit"
              disabled={busy || !online || !payload || !draft.trim()}
            >
              <DetailIcon />
            </IconBtn>
          </div>
        </form>
      </div>
    </div>
  )
}
