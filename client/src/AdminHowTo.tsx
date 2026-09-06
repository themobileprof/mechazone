/** Super admin: how-to cards for morphed AI playbook actions. */
import { useEffect, useMemo, useState } from 'react'
import { adminCreateHowTo, adminDeleteHowTo, adminListHowToActions, adminListHowTos, adminUpdateHowTo } from './api'
import { HowToEditor } from './HowToEditor'
import { BookIcon, DismissIcon, IconBtn, SaveIcon } from './chrome'
import { howtoPlateFiles } from './howto'
import type { HowToGuide, PlaybookAction } from './types'

const emptyForm = {
  title: '',
  blurb: '',
  warning: 'This card is a shop skill. It does not name pins on a module connector.',
  body_html: '<p></p>',
  match_words: '',
  published: true,
  action_ids: [] as string[],
}

export function AdminHowTo({ onError }: { onError: (msg: string | null) => void }) {
  const [guides, setGuides] = useState<HowToGuide[]>([])
  const [actions, setActions] = useState<PlaybookAction[]>([])
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [actionQ, setActionQ] = useState('')
  const [busy, setBusy] = useState(false)
  const plates = useMemo(() => howtoPlateFiles(), [])

  async function refresh() {
    const [g, a] = await Promise.all([adminListHowTos(), adminListHowToActions()])
    setGuides(g)
    setActions(a)
  }

  useEffect(() => {
    void refresh().catch((err: unknown) => onError(err instanceof Error ? err.message : String(err)))
  }, [onError])

  function load(g: HowToGuide) {
    setEditingId(g.id)
    setForm({
      title: g.title,
      blurb: g.blurb,
      warning: g.warning,
      body_html: g.body_html || '<p></p>',
      match_words: (g.match_words ?? []).join(', '),
      published: g.published,
      action_ids: g.action_ids ?? [],
    })
  }

  function startNew(from?: PlaybookAction) {
    setEditingId(null)
    setForm({
      ...emptyForm,
      title: from?.title ?? '',
      match_words: from ? from.title.toLowerCase() : '',
      action_ids: from ? [from.id] : [],
    })
  }

  const body = {
    title: form.title,
    blurb: form.blurb,
    warning: form.warning,
    body_html: form.body_html,
    match_words: form.match_words.split(',').map((w) => w.trim()).filter(Boolean),
    published: form.published,
    action_ids: form.action_ids,
  }

  async function save() {
    setBusy(true)
    onError(null)
    try {
      const saved = editingId ? await adminUpdateHowTo(editingId, body) : await adminCreateHowTo(body)
      await refresh()
      load(saved)
    } catch (err: unknown) {
      onError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!editingId) return
    if (!window.confirm('Delete this how-to card? The AI action list stays.')) return
    setBusy(true)
    onError(null)
    try {
      await adminDeleteHowTo(editingId)
      setEditingId(null)
      setForm(emptyForm)
      await refresh()
    } catch (err: unknown) {
      onError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const q = actionQ.trim().toLowerCase()
  const visibleActions = q
    ? actions.filter((a) => a.title.toLowerCase().includes(q) || a.kind.includes(q) || (a.variants ?? []).some((v) => v.toLowerCase().includes(q)))
    : actions

  return (
    <section className="flex h-full min-h-0 flex-col gap-3 md:flex-row">
      <div className="flex min-h-0 w-full flex-col border border-brass/20 bg-panel p-4 md:w-72 md:shrink-0">
        <h2 className="shrink-0 font-mono text-sm tracking-[0.25em] text-brass">CARDS</h2>
        <p className="mt-1 shrink-0 text-xs text-steel">Published cards show on HOW-TO. Similar AI steps share one action below.</p>
        <div className="mt-3 shrink-0">
          <IconBtn tip="Blank card" label="NEW CARD" tone="brass" onClick={() => startNew()}>
            <BookIcon />
          </IconBtn>
        </div>
        <ol className="mt-3 min-h-0 flex-1 space-y-1 overflow-y-auto overscroll-contain pr-1">
          {guides.map((g) => (
            <li key={g.id}>
              <button
                type="button"
                className={`w-full border px-3 py-2 text-left ${editingId === g.id ? 'border-brass/60 bg-brass/10' : 'border-steel/20'}`}
                onClick={() => load(g)}
              >
                <p className="font-semibold">{g.title}</p>
                <p className="font-mono text-[11px] text-steel">
                  {g.published ? 'LIVE' : 'DRAFT'} · {(g.match_words ?? []).length} words · {(g.action_ids ?? []).length} actions
                </p>
              </button>
            </li>
          ))}
        </ol>
      </div>

      <div className="flex min-h-0 min-w-0 flex-1 flex-col border border-brass/20 bg-panel p-4">
        <h2 className="shrink-0 font-mono text-sm tracking-[0.25em] text-brass">{editingId ? 'EDIT CARD' : 'NEW CARD'}</h2>
        <form
          className="mt-3 min-h-0 flex-1 space-y-3 overflow-y-auto overscroll-contain pr-1"
          onSubmit={(e) => { e.preventDefault(); void save() }}
        >
          <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Title" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required />
          <textarea className="min-h-16 w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Blurb" value={form.blurb} onChange={(e) => setForm({ ...form, blurb: e.target.value })} />
          <textarea className="min-h-16 w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Warning on the bay" value={form.warning} onChange={(e) => setForm({ ...form, warning: e.target.value })} />
          <input className="w-full border border-steel/30 bg-oil px-3 py-2" placeholder="Match words, comma-separated (ohm, volt, dlc)" value={form.match_words} onChange={(e) => setForm({ ...form, match_words: e.target.value })} />
          <label className="flex items-center gap-2 font-mono text-xs tracking-widest text-steel">
            <input type="checkbox" checked={form.published} onChange={(e) => setForm({ ...form, published: e.target.checked })} />
            PUBLISHED ON THE BAY
          </label>
          <HowToEditor key={editingId ?? 'new'} html={form.body_html} plates={plates} onChange={(body_html) => setForm((f) => ({ ...f, body_html }))} />
          <div>
            <p className="font-mono text-[11px] tracking-[0.2em] text-brass">AI ACTIONS</p>
            <p className="mt-1 text-xs text-steel">Playbook steps morph into these rows. Tick the ones this card teaches.</p>
            <input className="mt-2 w-full border border-steel/30 bg-oil px-3 py-2 text-sm" placeholder="Filter actions" value={actionQ} onChange={(e) => setActionQ(e.target.value)} />
            <ol className="mt-2 max-h-48 space-y-1 overflow-y-auto border border-steel/20 p-2">
              {visibleActions.length === 0 && <li className="font-mono text-xs text-steel">No actions yet — they appear after a playbook is built.</li>}
              {visibleActions.map((a) => {
                const on = form.action_ids.includes(a.id)
                return (
                  <li key={a.id} className="flex items-start gap-2">
                    <input
                      type="checkbox"
                      checked={on}
                      onChange={() => setForm((f) => ({
                        ...f,
                        action_ids: on ? f.action_ids.filter((id) => id !== a.id) : [...f.action_ids, a.id],
                      }))}
                    />
                    <button type="button" className="min-w-0 flex-1 text-left text-sm" onClick={() => startNew(a)}>
                      <span className="font-semibold">{a.title}</span>
                      <span className="block font-mono text-[11px] text-steel">{a.kind} · seen {a.seen_count}{a.variants?.length ? ` · ${a.variants.length} variants` : ''}</span>
                    </button>
                  </li>
                )
              })}
            </ol>
          </div>
          <div className="flex flex-wrap gap-2">
            <IconBtn tip={editingId ? 'Save this card' : 'Create this card'} label={busy ? 'SAVING' : 'SAVE'} tone="paper" type="submit" disabled={busy}>
              <SaveIcon />
            </IconBtn>
            {editingId && (
              <IconBtn tip="Delete this card. AI actions stay in the list." label="DELETE" tone="fault" disabled={busy} onClick={() => void remove()}>
                <DismissIcon />
              </IconBtn>
            )}
          </div>
        </form>
      </div>
    </section>
  )
}
