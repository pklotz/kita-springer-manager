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
