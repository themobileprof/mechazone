/** ISO 3779 / SAE J272 readout from the 17 characters. Not a vPIC replacement and not a diagnosis. */

export const TOYOTA_WMI = new Set([
  'JTD', 'JTE', 'JTN', 'JT2', 'JT3', 'JT4', 'JT5', 'JT6', 'JT8',
  '4T1', '4T3', '4T4', '5TD', '5TE', '5TF',
  'SB1', 'AHT', 'NMT', 'MR0', 'MR2', 'MM7', 'MM8', 'PN1', 'PN4',
  '2T1', '2T2', '2T3', '3TM', '4TA',
])

const LEXUS_WMI = new Set(['JTH', 'JTJ', 'JT8', '2T2', '58A'])

/** Public WMI prefixes → maker / plant country / body kind. Class lines stay conservative. */
const WMI: Record<string, { maker: string; region: string; kind?: string }> = {
  '2T1': { maker: 'Toyota', region: 'Canada', kind: 'passenger' },
  '2T2': { maker: 'Lexus', region: 'Canada', kind: 'passenger' },
  '2T3': { maker: 'Toyota', region: 'Canada' },
  '3TM': { maker: 'Toyota', region: 'Mexico' },
  '4T1': { maker: 'Toyota', region: 'USA', kind: 'passenger' },
  '4T3': { maker: 'Toyota', region: 'USA' },
  '4T4': { maker: 'Toyota', region: 'USA' },
  '4TA': { maker: 'Toyota', region: 'USA' },
  '5TD': { maker: 'Toyota', region: 'USA' },
  '5TE': { maker: 'Toyota', region: 'USA' },
  '5TF': { maker: 'Toyota', region: 'USA' },
  JTD: { maker: 'Toyota', region: 'Japan' },
  JTE: { maker: 'Toyota', region: 'Japan' },
  JTN: { maker: 'Toyota', region: 'Japan' },
  JT2: { maker: 'Toyota', region: 'Japan', kind: 'passenger' },
  JT3: { maker: 'Toyota', region: 'Japan' },
  JT4: { maker: 'Toyota', region: 'Japan' },
  JT5: { maker: 'Toyota', region: 'Japan' },
  JT6: { maker: 'Toyota', region: 'Japan' },
  JT8: { maker: 'Lexus', region: 'Japan' },
  SB1: { maker: 'Toyota', region: 'UK', kind: 'passenger' },
  AHT: { maker: 'Toyota', region: 'South Africa' },
  NMT: { maker: 'Toyota', region: 'Turkey' },
  MR0: { maker: 'Toyota', region: 'Thailand' },
  MR2: { maker: 'Toyota', region: 'Thailand' },
  MM7: { maker: 'Toyota', region: 'Thailand' },
  MM8: { maker: 'Toyota', region: 'Thailand' },
  PN1: { maker: 'Toyota', region: 'Malaysia' },
  PN4: { maker: 'Toyota', region: 'Malaysia' },
  JTH: { maker: 'Lexus', region: 'Japan' },
  JTJ: { maker: 'Lexus', region: 'Japan' },
  '1HG': { maker: 'Honda', region: 'USA', kind: 'passenger' },
  '2HG': { maker: 'Honda', region: 'Canada', kind: 'passenger' },
  '19X': { maker: 'Honda', region: 'USA', kind: 'passenger' },
  JHM: { maker: 'Honda', region: 'Japan', kind: 'passenger' },
  SHH: { maker: 'Honda', region: 'UK' },
}

const REGION_CHAR: Record<string, string> = {
  '1': 'USA',
  '4': 'USA',
  '5': 'USA',
  '2': 'Canada',
  '3': 'Mexico',
  J: 'Japan',
  K: 'Korea',
  S: 'UK',
  W: 'Germany',
  V: 'France',
  Z: 'Italy',
  L: 'China',
  T: 'Czechia / Switzerland',
  Y: 'Sweden / Finland',
  '6': 'Australia',
  '9': 'Brazil',
  A: 'South Africa',
  M: 'India',
  N: 'Turkey',
  H: 'China',
}

/** A=1980 … Y=2000, 1=2001 … 9=2009; then the 30-year cycle repeats (A=2010). Skip I,O,Q,U,Z. */
const YEAR_CYCLE = 'ABCDEFGHJKLMNPRSTVWXY123456789'

const CHECK_MAP: Record<string, number> = {
  A: 1, B: 2, C: 3, D: 4, E: 5, F: 6, G: 7, H: 8,
  J: 1, K: 2, L: 3, M: 4, N: 5, P: 7, R: 9, S: 2,
  T: 3, U: 4, V: 5, W: 6, X: 7, Y: 8, Z: 9,
  '0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
}
const CHECK_WEIGHTS = [8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2]

export type VinPlate = {
  vin: string
  wmi: string
  yearCode: string
  year: number | null
  plant: string
  maker: string
  region: string
  kind: string
  classLine: string
  busNote: string
  checkOk: boolean | null
  /** Large line: year + maker, or year + maker + class when that is all we have. */
  headline: string
  /** One sentence under the headline. */
  factLine: string
  chips: string[]
}

export function namedModel(name: string | undefined) {
  const m = (name || '').trim()
  return m !== '' && m.toLowerCase() !== 'unknown'
}

export function modelYearFromVin(vin: string, now = new Date().getFullYear()): number | null {
  if (vin.length < 10) return null
  const i = YEAR_CYCLE.indexOf(vin[9]!)
  if (i < 0) return null
  const cap = now + 1
  const years = [1980 + i, 2010 + i, 2040 + i].filter((y) => y <= cap)
  return years.length ? years[years.length - 1]! : null
}

export function vinCheckOk(vin: string): boolean | null {
  if (vin.length !== 17) return null
  let sum = 0
  for (let i = 0; i < 17; i++) {
    const n = CHECK_MAP[vin[i]!]
    if (n === undefined) return null
    sum += n * CHECK_WEIGHTS[i]!
  }
  const rem = sum % 11
  const expect = rem === 10 ? 'X' : String(rem)
  return vin[8] === expect
}

function toyotaClass(wmi: string, year: number | null): string {
  if (wmi !== '2T1' || !year) return ''
  if (year >= 1998 && year <= 2002) return '8th-gen Corolla class'
  if (year >= 2003 && year <= 2008) return '9th-gen Corolla / Matrix class'
  if (year >= 2009 && year <= 2013) return '10th-gen Corolla / Matrix class'
  if (year >= 2014 && year <= 2019) return '11th-gen Corolla class'
  if (year >= 2020 && year <= 2026) return '12th-gen Corolla class'
  return ''
}

function busNote(maker: string, year: number | null): string {
  const toyota = maker === 'Toyota' || maker === 'Lexus'
  if (toyota && year && year <= 2007) {
    return 'This era often talks K-Line (DLC pin 7). This bay’s OpenPort path opens 500 kbit/s ISO-TP/CAN — a silent scan here is not a Toyota lockout.'
  }
  return ''
}

export function readVinPlate(vin: string, now = new Date().getFullYear()): VinPlate | null {
  const v = vin.trim().toUpperCase()
  if (v.length !== 17) return null
  const wmi = v.slice(0, 3)
  const known = WMI[wmi]
  const maker = known?.maker || (TOYOTA_WMI.has(wmi) ? 'Toyota' : LEXUS_WMI.has(wmi) ? 'Lexus' : '')
  const region = known?.region || REGION_CHAR[v[0]!] || ''
  const kind = known?.kind || ''
  const yearCode = v[9]!
  const year = modelYearFromVin(v, now)
  const plant = v[10]!
  const classLine = toyotaClass(wmi, year)
  const checkOk = vinCheckOk(v)
  const yearBit = year ? String(year) : ''
  const headline = [yearBit, maker || (region ? `${region} vehicle` : 'VIN plate')].filter(Boolean).join(' ')
  const who = [maker && region ? `${maker} ${region}` : maker || region, kind].filter(Boolean).join(' ')
  const yearFact = year
    ? `year digit ${yearCode} is a ${year} model`
    : `year digit ${yearCode} is not an SAE model-year code`
  const factLine = [who, yearFact, classLine].filter(Boolean).join(' · ')
  const chips = [
    `WMI ${wmi}`,
    plant ? `plant ${plant}` : '',
    checkOk === true ? 'check digit OK' : checkOk === false ? 'check digit mismatch — re-read the plate' : '',
  ].filter(Boolean)
  return {
    vin: v,
    wmi,
    yearCode,
    year,
    plant,
    maker,
    region,
    kind,
    classLine,
    busNote: busNote(maker, year),
    checkOk,
    headline,
    factLine,
    chips,
  }
}

export function displayHeadline(
  plate: VinPlate,
  vehicle?: { vin?: string; make?: string; model?: string; manufacture_year?: number } | null,
): { kicker: string; title: string; fromDecode: boolean } {
  const sameVin = Boolean(vehicle?.vin) && vehicle?.vin === plate.vin
  const year = sameVin && vehicle?.manufacture_year && vehicle.manufacture_year > 0
    ? vehicle.manufacture_year
    : plate.year
  const make = sameVin && namedModel(vehicle?.make) ? vehicle!.make : plate.maker
  if (sameVin && namedModel(vehicle?.model)) {
    return {
      kicker: 'DECODE',
      title: [year || '', make, vehicle!.model].filter(Boolean).join(' '),
      fromDecode: true,
    }
  }
  return {
    kicker: 'FROM THE VIN',
    title: plate.headline,
    fromDecode: false,
  }
}
