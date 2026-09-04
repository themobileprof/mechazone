/** WMI → probe class for the bay. Same prefixes as the worker; not a diagnosis. */
const TOYOTA_WMI = new Set([
  'JTD', 'JTE', 'JTN', 'JT2', 'JT3', 'JT4', 'JT5', 'JT6', 'JT8',
  '4T1', '4T3', '4T4', '5TD', '5TE', '5TF',
  'SB1', 'AHT', 'NMT', 'MR0', 'MR2', 'MM7', 'MM8', 'PN1', 'PN4',
  '2T1', '2T2', '2T3', '3TM', '4TA',
])

export function toyotaClassVin(vin: string): boolean {
  return TOYOTA_WMI.has(vin.trim().toUpperCase().slice(0, 3))
}

export function kitVinGap(text: string): boolean {
  const t = text.toLowerCase()
  return t.includes('f190') || t.includes('kit vin') || (t.includes('did not answer') && t.includes('vin'))
}

export function probePreview(opts: {
  vin: string
  model?: string
  bookModel?: string
  profile?: string
}): { id: string; label: string } {
  const model = (opts.bookModel || opts.model || '').toLowerCase()
  if (opts.profile === 'avensis_3zr_fae' || model.startsWith('avensis')) {
    return { id: 'avensis_3zr_fae', label: 'Captured Avensis map' }
  }
  if (opts.profile === 'toyota_common' || toyotaClassVin(opts.vin)) {
    return { id: 'toyota_common', label: 'Toyota 11-bit probe' }
  }
  if (opts.profile === 'generic_uds') {
    return { id: 'generic_uds', label: 'ISO 15765-4 probe (7E0–7E2)' }
  }
  if (opts.vin.length === 17) {
    return { id: 'generic_uds', label: 'ISO 15765-4 probe (7E0–7E2)' }
  }
  return { id: '', label: 'Type the VIN to pick a probe — the kit rarely fills it' }
}
