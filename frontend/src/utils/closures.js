export const closureLabel = (c) => {
  if (c.type === 'springerin') return 'Urlaub'
  if (c.type === 'provider') return 'Schliesstag Träger'
  if (c.type === 'kita') return 'Schliesstag Kita'
  return c.type
}
