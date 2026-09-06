import { useEffect, useId, useRef, useState } from 'react'
import { howtoSrc, type HowToGuide as FallbackHowTo } from './howto'
import type { HowToGuide } from './types'

export type HowToCard = Pick<HowToGuide, 'id' | 'title' | 'blurb' | 'warning'> & {
  body_html?: string
  steps?: FallbackHowTo['steps']
}

function Plate({ file, alt, hunt }: { file: string; alt: string; hunt: string }) {
  const [missing, setMissing] = useState(false)
  const src = howtoSrc(file)
  return (
    <figure className="howto-plate">
      {missing ? (
        <div className="howto-missing">
          <p className="font-mono text-[11px] tracking-[0.25em] text-brass">PLATE NOT ON DISK</p>
          <p className="mt-2 text-sm text-steel">{hunt}</p>
          <p className="mt-2 font-mono text-xs text-steel">{file}</p>
        </div>
      ) : (
        <img
          alt={alt}
          className={`howto-img${file.endsWith('.svg') ? ' is-diagram' : ''}`}
          src={src}
          onError={() => setMissing(true)}
        />
      )}
    </figure>
  )
}

export function HowToModal({
  guides,
  onClose,
}: {
  guides: HowToCard[]
  onClose: () => void
}) {
  const titleId = useId()
  const closeRef = useRef<HTMLButtonElement>(null)
  const [active, setActive] = useState(guides[0]?.id ?? '')
  const guide = guides.find((g) => g.id === active) ?? guides[0]

  useEffect(() => {
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  if (!guide) return null

  return (
    <div className="howto-overlay" onClick={onClose} role="presentation">
      <div
        className="howto-sheet"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-brass/25 pb-3">
          <div>
            <p className="font-mono text-[11px] tracking-[0.3em] text-brass">HOW TO · BAY CARD</p>
            <h2 id={titleId} className="mt-1 font-poster text-2xl tracking-wide text-paper">
              {guide.title}
            </h2>
            <p className="mt-1 max-w-xl text-sm text-steel">{guide.blurb}</p>
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
        {guides.length > 1 && (
          <div className="mt-3 flex flex-wrap gap-1">
            {guides.map((g) => (
              <button
                key={g.id}
                type="button"
                className={`min-h-10 border px-3 font-mono text-[11px] tracking-widest ${g.id === guide.id ? 'border-brass bg-brass/15 text-brass' : 'border-steel/30 text-steel'}`}
                onClick={() => setActive(g.id)}
              >
                {g.title}
              </button>
            ))}
          </div>
        )}
        <p className="mt-3 border border-brass/40 bg-brass/10 px-3 py-2 text-sm">{guide.warning}</p>
        {guide.body_html ? (
          <div className="howto-body mt-4" dangerouslySetInnerHTML={{ __html: guide.body_html }} />
        ) : (
          <ol className="mt-4 space-y-5">
            {(guide.steps ?? []).map((st, i) => (
              <li key={st.file} className="border-l-2 border-brass/40 pl-3">
                <p className="font-mono text-[11px] tracking-[0.25em] text-brass">
                  {String(i + 1).padStart(2, '0')} · {st.title.toUpperCase()}
                </p>
                <p className="mt-1 text-sm text-steel">{st.body}</p>
                <Plate file={st.file} alt={st.alt} hunt={st.hunt} />
              </li>
            ))}
          </ol>
        )}
      </div>
    </div>
  )
}
