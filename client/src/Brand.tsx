/** Wordless shop mark (car + headset). Pair with a kicker; do not restyle the PNG. */

export function Logo({ className = 'h-12 w-auto md:h-14' }: { className?: string }) {
  return (
    <img
      src="/mechazone.png"
      alt="Mechazone"
      className={className}
      draggable={false}
    />
  )
}
