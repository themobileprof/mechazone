/** Shop-floor chrome: stencil icons, hover/focus tooltips, labeled icon buttons. */
import type { ReactNode } from 'react'

export function Tip({ label, children }: { label: string; children: ReactNode }) {
  return (
    <span className="tip">
      {children}
      <span className="tip-bubble" role="tooltip">{label}</span>
    </span>
  )
}

type Tone = 'ghost' | 'brass' | 'paper' | 'ok' | 'fault'

export function IconBtn({
  tip,
  label,
  children,
  onClick,
  disabled,
  tone = 'ghost',
  type = 'button',
}: {
  tip: string
  label?: string
  children: ReactNode
  onClick?: () => void
  disabled?: boolean
  tone?: Tone
  type?: 'button' | 'submit'
}) {
  const tones: Record<Tone, string> = {
    ghost: 'border-paper/40 text-paper hover:border-brass hover:text-brass',
    brass: 'border-brass bg-brass text-oil font-semibold',
    paper: 'border-paper bg-paper text-oil font-semibold',
    ok: 'border-ok text-ok hover:bg-ok/10',
    fault: 'border-fault/50 text-steel hover:border-fault hover:text-fault',
  }
  return (
    <span className="tip">
      <button
        type={type}
        className={`icon-btn ${label ? 'icon-btn-labeled' : ''} ${tones[tone]} disabled:opacity-40`}
        aria-label={tip}
        title={tip}
        disabled={disabled}
        onClick={onClick}
      >
        {children}
        {label ? <span className="font-mono text-[11px] tracking-widest">{label}</span> : null}
      </button>
      <span className="tip-bubble" role="tooltip">{tip}</span>
    </span>
  )
}

function Glyph({ children }: { children: ReactNode }) {
  return (
    <svg viewBox="0 0 24 24" className="glyph" aria-hidden>
      {children}
    </svg>
  )
}

const stroke = {
  fill: 'none' as const,
  stroke: 'currentColor',
  strokeWidth: 1.6,
  strokeLinecap: 'square' as const,
  strokeLinejoin: 'miter' as const,
}

export function PlugIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M7 3v5M17 3v5M5 8h14v6H5zM9 14v7M15 14v7" />
    </Glyph>
  )
}

export function RefreshIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M4 12a8 8 0 0 1 13.5-5.5L20 9" />
      <path {...stroke} d="M20 4v5h-5M20 12a8 8 0 0 1-13.5 5.5L4 15" />
      <path {...stroke} d="M4 20v-5h5" />
    </Glyph>
  )
}

export function LinkIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M9 12H4V8h5M15 12h5v4h-5M9 10v4M15 10v4" />
      <path {...stroke} d="M9 12h6" />
    </Glyph>
  )
}

export function ChassisIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M3 14h18M6 14V9l3-3h6l3 3v5M7 17h2M15 17h2" />
    </Glyph>
  )
}

export function ScanIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M5 7h14v10H5zM8 10h8M8 13h5" />
      <path {...stroke} d="M3 12h2M19 12h2" />
    </Glyph>
  )
}

export function AttachIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M8 7h8l3 3v9H8z" />
      <path {...stroke} d="M16 7v4h4M10 13h6M10 16h4" />
    </Glyph>
  )
}

export function MeterIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M6 9h12v11H6zM9 9V6h6v3M12 13l3 2" />
    </Glyph>
  )
}

export function DetailIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M6 4h9l5 5v11H6zM15 4v5h5M8 12h8M8 15h5" />
    </Glyph>
  )
}

export function BookIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M4 5h7v14H5a1 1 0 0 1-1-1zM20 5h-7v14h6a1 1 0 0 0 1-1zM12 5v14" />
    </Glyph>
  )
}

export function StampIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M7 4h10v4H7zM5 12h14v4H5zM8 16v4h8v-4" />
    </Glyph>
  )
}

export function FolderIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M3 7h6l2 2h10v10H3z" />
    </Glyph>
  )
}

export function WrenchIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M14 6l4 4-8 8-4-4zM16 4l2 2M6 16l-2 4" />
    </Glyph>
  )
}

export function DismissIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M6 6l12 12M18 6L6 18" />
    </Glyph>
  )
}

export function SignOutIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M5 5h8v14H5zM13 12h7M17 9l3 3-3 3" />
    </Glyph>
  )
}

export function LockIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M8 11V8a4 4 0 0 1 8 0v3M6 11h12v9H6z" />
    </Glyph>
  )
}

export function CheckIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M5 13l4 4 10-10" />
    </Glyph>
  )
}

export function ChevronLeftIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M14 6l-6 6 6 6" />
    </Glyph>
  )
}

export function ChevronRightIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M10 6l6 6-6 6" />
    </Glyph>
  )
}

export function QueueIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M5 6h14M5 12h14M5 18h9" />
    </Glyph>
  )
}

export function ShopIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M4 10l2-5h12l2 5M4 10h16v9H4zM10 19v-5h4v5" />
    </Glyph>
  )
}

export function LoginIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M12 4a4 4 0 1 1 0 8 4 4 0 0 1 0-8zM5 20c1.5-3 4-5 7-5s5.5 2 7 5" />
    </Glyph>
  )
}

export function WaveIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M3 12h3l2-5 3 10 3-8 2 3h5" />
    </Glyph>
  )
}

export function SaveIcon() {
  return (
    <Glyph>
      <path {...stroke} d="M5 5h11l3 3v11H5zM8 5v4h8M8 14h8" />
    </Glyph>
  )
}
