// diffMinutes returns the difference between two HH:MM strings in minutes.
// Handles overnight spans (end < start → +24h). Returns 0 if either value is missing.
export function diffMinutes(start, end) {
  if (!start || !end) return 0
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  let mins = (eh * 60 + em) - (sh * 60 + sm)
  if (mins < 0) mins += 24 * 60
  return mins
}

export function formatHours(minutes) {
  if (!minutes) return ''
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (m === 0) return String(h)
  return (h + m / 60).toFixed(2).replace(/\.?0+$/, '')
}

// formatHm returns e.g. "9:00" or "0:30" for compact duration display.
export function formatHm(minutes) {
  if (!minutes || minutes <= 0) return '0:00'
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return `${h}:${String(m).padStart(2, '0')}`
}

// breakMinutes returns the break duration in minutes from the stored integer.
export function breakMinutes(breakMin) {
  return breakMin || 0
}

// netWorkMinutes returns net working time: (end − start) − breakMin.
export function netWorkMinutes(start, breakMin, end) {
  if (!start || !end) return 0
  return Math.max(0, diffMinutes(start, end) - (breakMin || 0))
}

// grossWorkMinutes returns total elapsed time between start and end, regardless
// of break. Used to derive the legal minimum-break threshold.
export function grossWorkMinutes(start, end) {
  return diffMinutes(start, end)
}

// legalMinBreakMinutes returns the statutory minimum break for a given gross
// shift length, per Swiss Arbeitsgesetz Art. 15 Abs. 1:
//   >5.5 h → 15 min; >7 h → 30 min; >9 h → 60 min.
export function legalMinBreakMinutes(grossMin) {
  if (grossMin > 9 * 60) return 60
  if (grossMin > 7 * 60) return 30
  if (grossMin > 5.5 * 60) return 15
  return 0
}

// requiredBreakMinutes returns max(legal, provider default) for a given shift.
export function requiredBreakMinutes(grossMin, providerMin = 0) {
  return Math.max(legalMinBreakMinutes(grossMin), providerMin || 0)
}
