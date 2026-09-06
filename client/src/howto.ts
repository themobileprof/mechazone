/** Generic shop skills for playbook steps. These cards do not name pins on a vehicle. */

export type HowToStep = {
  title: string
  body: string
  file: string
  alt: string
  hunt: string
}

export type HowToGuide = {
  id: string
  title: string
  blurb: string
  warning: string
  match: RegExp
  steps: HowToStep[]
}

function plate(file: string, alt: string, hunt: string): Pick<HowToStep, 'file' | 'alt' | 'hunt'> {
  return { file, alt, hunt }
}

export const HOWTOS: HowToGuide[] = [
  {
    id: 'dlc-power-ground',
    title: 'Power and ground at the 16-pin DLC',
    blurb: 'The under-dash OBD port is SAE J1962. Pin 16 is battery. Pins 4 and 5 are grounds. That numbering is the port, not a random ECU plug.',
    warning: 'Use this card only when the playbook means the diagnostic link connector under the dash. If it named a module connector, ignore DLC pin numbers and use the cited figure.',
    match: /\bdlc\b|obd-?ii|16-pin|diagnostic link|\bpin\s*16\b|j1962/i,
    steps: [
      {
        title: 'This is the OBD2 port',
        body: 'The socket in the car is a female trapezoid: two rows of eight holes, plastic housing, often dusty. It is not a round cigarette-lighter hole and not a small OEM plug beside it. If it does not look like this, you are on the wrong connector.',
        ...plate(
          'obd2-port-face.jpg',
          'Close-up of a 16-pin female OBD-II socket in a vehicle',
          'Straight-on photo of the female 16-pin DLC (empty cavities, no pins sticking out).',
        ),
      },
      {
        title: 'Find it under the dash',
        body: 'Sit in the driver seat. The 16-pin DLC is usually under the dash, left of the steering column, within arm’s reach. Do not confuse it with a smaller OEM plug next to it.',
        ...plate(
          'dlc-power-ground-location.jpg',
          '16-pin diagnostic port under a dashboard, no number plate in frame',
          'Wide shot from the driver footwell: trapezoid 16-pin DLC in shadow under the dash, steering column visible, no licence plate or VIN tag.',
        ),
      },
      {
        title: 'Know which holes are which',
        body: 'Look into the car socket with the wide side up. Top row is pins 1–8 left to right. Bottom row is 9–16. Pin 16 (bottom right) is battery. Pin 4 is chassis ground. Pin 5 is signal ground. If the playbook said “pin 4 or 5” on the DLC, those are the two grounds. Do not use these numbers on a module connector.',
        ...plate(
          'dlc-power-ground-pinout.svg',
          'SAE J1962 vehicle socket pin map with 4, 5, and 16 marked',
          'Female DLC pin map, looking in, 1–8 over 9–16.',
        ),
      },
      {
        title: 'Prove battery on pin 16',
        body: 'Key as the playbook says (usually ignition on). Meter on DC volts. Black probe on battery negative or a known-good chassis ground. Red probe into pin 16. Expect about battery voltage (often 12–14 V). Zero or OL means the port is not powered — stop and fix that before any module test.',
        ...plate(
          'dlc-power-ground-pin16.jpg',
          'Red probe in DLC pin 16, meter showing about 12 volts',
          'Hands: red backprobe in the bottom-right DLC cavity, black on a battery negative clamp, meter display about 12.6 V. No faces.',
        ),
      },
    ],
  },
  {
    id: 'backprobe',
    title: 'Land on a pin without wrecking the connector',
    blurb: 'Most live tests want the connector still plugged in. You reach the metal from the wire side, not by stabbing the insulation.',
    warning: 'The playbook (or a cited figure) names the cavity. This card does not. Do not pierce the wire if you have a backprobe pin or a meter lead that fits the rear of the housing.',
    match: /back-?probe|from the (back|rear) of the connector|cavity|still plugged/i,
    steps: [
      {
        title: 'Leave it connected',
        body: 'Push the connector fully home so the lock clicks. You are measuring the circuit as the car sees it. Unplugging is for ohms on a component, not for a live voltage at a running module.',
        ...plate(
          'backprobe-connected.jpg',
          'A locked automotive connector still mated, wires exiting the back',
          'Close-up of a generic weather-pack style connector fully seated, CPA lock down, loom exiting the rear. No OEM logo readable.',
        ),
      },
      {
        title: 'Slide in from the wire side',
        body: 'From the back of the housing, slip a backprobe pin or a thin meter probe alongside the wire until it touches the terminal. You should feel a light stop. Do not force it past the seal any harder than a paperclip into a tight hole.',
        ...plate(
          'backprobe-rear.jpg',
          'A thin probe entering the rear of a connector beside a wire',
          'Macro: gold or nickel backprobe needle sliding into a sealed cavity next to one wire, connector still mated. Clean background.',
        ),
      },
      {
        title: 'Do not pierce the loom',
        body: 'A vampire / piercing probe through the insulation lets water in and can cut a strand. Use it only if you have no rear access and you will seal the nick. Prefer the rear cavity.',
        ...plate(
          'backprobe-no-pierce.jpg',
          'A piercing probe on a wire marked as the wrong method',
          'Split or single frame: a piercing clip biting through insulation (this is the “avoid” shot). Neutral workbench, no brand.',
        ),
      },
    ],
  },
  {
    id: 'meter-continuity',
    title: 'Continuity (beep) — is there a path?',
    blurb: 'Power off. You are asking: do these two points connect through a wire or a closed switch? A beep is a path. Silence plus OL is open.',
    warning: 'Never continuity-test a live circuit. Key out, battery still in the car is fine, but no voltage on the two points. If the playbook named two pins, those are the two points — not “any pin 4”.',
    match: /continuit|beep test|\bdiode mode\b|open (wire|circuit)/i,
    steps: [
      {
        title: 'Leads and dial',
        body: 'Black lead in COM. Red lead in the jack labelled VΩmA or VΩ (not the 10 A / 20 A jack). Turn the dial to the speaker / diode symbol — not V, not A, not a high ohm range if your meter has a dedicated beep.',
        ...plate(
          'meter-continuity-dial.jpg',
          'Multimeter dial on the continuity or diode symbol',
          'Top-down meter: dial click-stopped on the speaker or diode mark, black in COM, red in VΩ. Typical yellow/black or red/black shop meter.',
        ),
      },
      {
        title: 'Prove the meter first',
        body: 'Touch the two probe tips together. You should hear a beep and see a number near 0 Ω (maybe 0.2–0.5 because of the leads). If nothing happens, the leads are in the wrong holes or the dial is wrong.',
        ...plate(
          'meter-continuity-beep.jpg',
          'Probe tips touching, meter beeping near zero ohms',
          'Two probe tips in contact, meter display 0.3 Ω, optional speaker icon. Bench, no car needed.',
        ),
      },
      {
        title: 'Then the two points from the playbook',
        body: 'One probe on each point the playbook named. Beep + low ohms = a path. No beep and OL or 1. on the left of the display = open. A few ohms on a long earth strap can still be good; a few thousand on a supposed short jumper is not.',
        ...plate(
          'meter-continuity-ol.jpg',
          'Meter display showing OL with probes on an open circuit',
          'Meter face filling the frame, letters OL (or 1. on the left), probes not touching. Sharp, readable digits.',
        ),
      },
    ],
  },
  {
    id: 'meter-ohms',
    title: 'Measure resistance (ohms)',
    blurb: 'Power off. Red and black go in specific holes. The dial must be on Ω. The number is how hard it is for current to get through. OL means the path is open.',
    warning: 'Do not measure ohms on a live pin. Key out. Unplug the component if the playbook says to. The pin numbers come from the playbook or a cited figure — this card never invents pin 4 or pin 5 on a mystery connector.',
    match: /ohm|resist|Ω|kΩ|mΩ|kohm|megohm/i,
    steps: [
      {
        title: 'Plug the leads in',
        body: 'Black banana plug → COM (common). Red banana plug → the jack labelled VΩmA, VΩ, or a star of V / Ω / mA. Do not put red in the separate 10 A jack — that hole is only for high current, and ohms will not work there.',
        ...plate(
          'meter-ohms-jacks.jpg',
          'Black lead in COM and red lead in the V ohms milliamps jack',
          'Tight crop of meter input jacks: black fully seated in COM, red in VΩmA, the 10A jack empty. Labels readable.',
        ),
      },
      {
        title: 'Set the dial to Ω',
        body: 'Turn to the Greek omega (Ω). If the meter is manual-range, start on a middle range (200 Ω or 2 kΩ) and go up if you only see OL. Auto-range meters pick the range for you. Not V (volts). Not A (amps). Not the speaker unless the playbook asked for continuity.',
        ...plate(
          'meter-ohms-dial.jpg',
          'Multimeter rotary switch pointing at the ohms symbol',
          'Dial pointed at Ω, not V or A. Finger on the knob optional. Same meter family as the jacks shot if you can.',
        ),
      },
      {
        title: 'Zero the leads',
        body: 'Touch red and black tips together. You should see a small number, often 0.2–0.8 Ω. That is the leads, not the car. If you see OL, the dial or the jacks are still wrong.',
        ...plate(
          'meter-ohms-zero.jpg',
          'Probes shorted together, display a fraction of an ohm',
          'Probe tips touching, display 0.4 Ω. Same meter. Well lit digits.',
        ),
      },
      {
        title: 'Read the two points the playbook named',
        body: 'One probe on each pin or terminal in the playbook (for example the two ends of a sensor, or a pin to ground if it said so). Hold still. OL or 1. on the left = open (no path). A number with k means thousands of ohms (2.20 kΩ is 2200 Ω). M means millions. Compare to the pass/fail line on that playbook step — not to a number from this card.',
        ...plate(
          'meter-ohms-on-pins.jpg',
          'Probes on two terminals of an unplugged generic connector',
          'Unplugged generic connector on a rag: two meter probes in two cavities. No Toyota/Honda stamp readable. Display shows a few kΩ or OL — either is fine; we want the hand position.',
        ),
      },
    ],
  },
  {
    id: 'meter-volts',
    title: 'Measure DC voltage',
    blurb: 'This is how much electrical “pressure” is on a pin relative to ground. Key position follows the playbook (often ignition on, engine off).',
    warning: 'Use DC volts (solid line), not AC (wavy line). Black goes on a ground the playbook named (battery negative or chassis). Red goes on the test pin. This card does not pick the pin.',
    match: /\bvolts?\b|voltage|\bvdc\b|dc volts?/i,
    steps: [
      {
        title: 'Leads: same holes as ohms',
        body: 'Black in COM. Red in VΩmA. The 10 A jack stays empty. Voltage on the 10 A jack can blow the meter’s fuse or read nonsense.',
        ...plate(
          'meter-volts-jacks.jpg',
          'Black in COM, red in the voltage jack',
          'Same style as the ohms jacks photo: COM + VΩ, 10A empty. You may reuse one photo for both cards if the labels are identical.',
        ),
      },
      {
        title: 'Dial to V with a solid line',
        body: 'That is DC. The wavy line is AC mains — wrong for a 12 V car. Auto-range is fine. Manual-range: 20 V DC covers battery and 5 V sensor supplies.',
        ...plate(
          'meter-volts-dial.jpg',
          'Dial on DC volts, not AC',
          'Dial on V⎓ or V with a solid bar. The AC V~ position is visible but not selected.',
        ),
      },
      {
        title: 'Black on ground, then red on the pin',
        body: 'Clip or hold black on battery negative or clean chassis (paint is an insulator). Red on the pin or backprobe the playbook named. Hold still. Typical pictures: about 12–14 V is battery, about 5 V is often a sensor supply, near 0 V is ground or a dead feed. OL means the probe is not on metal. Write the number you see in the finding — do not invent a spec.',
        ...plate(
          'meter-volts-display.jpg',
          'Meter showing about 12 volts with probes on a battery',
          'Meter display ~12.4 V, black on a battery negative post, red on battery positive for a “known good” shot. Then you know what a real 12 V looks like on that meter.',
        ),
      },
    ],
  },
]

export function matchHowTos(title: string, detail: string, kind?: string): HowToGuide[] {
  const blob = `${title} ${detail} ${kind ?? ''}`
  return HOWTOS.filter((g) => g.match.test(blob))
}

export function matchCatalog(
  title: string,
  detail: string,
  kind: string | undefined,
  catalog: import('./types').HowToGuide[],
  howtoIds?: string[],
): import('./types').HowToGuide[] {
  const blob = `${title} ${detail} ${kind ?? ''}`.toLowerCase()
  const ids = new Set(howtoIds ?? [])
  return catalog.filter((g) => {
    if (g.published === false) return false
    if (ids.has(g.id)) return true
    return (g.match_words ?? []).some((w) => w && blob.includes(w.toLowerCase()))
  })
}

export function howtoPlateFiles() {
  return [...new Set(HOWTOS.flatMap((g) => g.steps.map((s) => s.file)))].sort()
}

export function howtoSrc(file: string) {
  return `/howto/${file}`
}
