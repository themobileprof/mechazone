import { displayHeadline, namedModel, readVinPlate } from './vinReadout'

type VehicleBits = {
  vin?: string
  make?: string
  model?: string
  manufacture_year?: number
  decode_source?: string
} | null | undefined

export function VinIdentity({
  vin,
  vehicle,
  bookLabel,
  probeLabel,
  variant = 'full',
}: {
  vin: string
  vehicle?: VehicleBits
  bookLabel?: string
  probeLabel?: string
  variant?: 'full' | 'compact' | 'rail'
}) {
  const plate = readVinPlate(vin)
  if (!plate) return null
  const sameVin = Boolean(vehicle?.vin) && vehicle!.vin === plate.vin
  const shown = displayHeadline(plate, sameVin ? vehicle : null)
  const decodeNote = shown.fromDecode
    ? (vehicle?.decode_source ? `Ledger ${vehicle.decode_source}` : 'Named by decode')
    : sameVin && namedModel(vehicle?.make)
      ? `${vehicle!.make} — decode did not name the body`
      : 'vPIC did not name this VIN — this is from the 17 characters'

  if (variant === 'rail') {
    return (
      <p className="vin-rail" aria-live="polite">
        <span className="vin-rail-title">{shown.title}</span>
        {plate.classLine ? <span className="vin-rail-class">{plate.classLine}</span> : null}
      </p>
    )
  }

  const compact = variant === 'compact'
  return (
    <aside className={`vin-plate ${compact ? 'vin-plate-compact' : ''}`} aria-live="polite">
      <p className="vin-plate-kicker">{shown.kicker}</p>
      <p className="vin-plate-title">{shown.title}</p>
      <p className="vin-plate-vin">{plate.vin}</p>
      <p className="vin-plate-facts">{plate.factLine}</p>
      {plate.busNote ? <p className="vin-plate-bus">{plate.busNote}</p> : null}
      <ul className="vin-plate-chips">
        {plate.chips.map((c) => (
          <li key={c}>{c}</li>
        ))}
        {probeLabel ? <li>{probeLabel}</li> : null}
        {bookLabel ? <li>{bookLabel}</li> : null}
        <li>{decodeNote}</li>
      </ul>
    </aside>
  )
}
