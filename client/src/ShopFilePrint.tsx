/** Print / Save-as-PDF sheet of this shop's jobs on one VIN. Hidden on screen. */
import { Logo } from './Brand'
import type { HistoryResponse, Principal } from './types'

function stamp(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}

function shopLine(user: Principal) {
  if (user.freelancer) return user.technician_name ? `Freelancer · ${user.technician_name}` : 'Freelancer'
  return user.shop_name || 'Shop'
}

export function ShopFilePrint({
  user,
  vin,
  history,
}: {
  user: Principal
  vin: string
  history: HistoryResponse
}) {
  const shop = shopLine(user)
  const vehicle = history.vehicle
  const plate = history.customer?.plate?.trim()
  const jobs = history.jobs ?? []
  const printed = new Date().toLocaleString()

  return (
    <article className="shop-file-print" aria-hidden>
      <header className="shop-file-print-head">
        <Logo className="shop-file-print-logo" />
        <div>
          <p className="shop-file-print-kicker">This shop&apos;s work</p>
          <h1>{shop}</h1>
          <p>Printed {printed}. Not a public vehicle history.</p>
        </div>
      </header>

      <dl className="shop-file-print-meta">
        <div>
          <dt>VIN</dt>
          <dd className="shop-file-print-vin">{vin}</dd>
        </div>
        {(vehicle?.make || vehicle?.model || vehicle?.manufacture_year) && (
          <div>
            <dt>Vehicle</dt>
            <dd>
              {[vehicle?.make, vehicle?.model, vehicle?.manufacture_year || null].filter(Boolean).join(' ')}
            </dd>
          </div>
        )}
        {plate && (
          <div>
            <dt>Plate</dt>
            <dd>{plate}</dd>
          </div>
        )}
      </dl>

      {jobs.length === 0 ? (
        <p className="shop-file-print-empty">
          {history.capture
            ? 'This shop has scanned this vehicle. No job has been closed yet.'
            : 'This shop has no jobs on this VIN.'}
        </p>
      ) : (
        <ol className="shop-file-print-jobs">
          {jobs.map((job) => (
            <li key={job.session_id}>
              <p className="shop-file-print-when">
                {stamp(job.created_at)}
                {job.technician_name ? ` · ${job.technician_name}` : ''}
                {job.mileage_km ? ` · ${job.mileage_km} km` : ''}
                {job.outcome ? ` · ${job.outcome}` : ''}
                {job.verified_fix ? ' · verified closeout' : ''}
                {job.import ? ` · attached ${job.import.source} report` : ''}
              </p>
              {job.work ? <p>{job.work}</p> : job.import?.note ? <p>{job.import.note}</p> : (
                <p className="shop-file-print-muted">Scan logged — no closeout written.</p>
              )}
              {job.parts_replaced.length > 0 && (
                <p>Parts: {job.parts_replaced.join(', ')}</p>
              )}
              {(job.closeout_code || job.active_codes.length > 0) && (
                <p className="shop-file-print-codes">
                  {job.closeout_code || job.active_codes.join('  ')}
                </p>
              )}
            </li>
          ))}
        </ol>
      )}

      <footer className="shop-file-print-foot">
        <p>
          This sheet is {shop}&apos;s file on this VIN. Another workshop cannot look it up by VIN.
          The owner may hand this copy to a shop they choose. Customer phone is not included.
        </p>
      </footer>
    </article>
  )
}
