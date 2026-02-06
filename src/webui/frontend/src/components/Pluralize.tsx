// Auto-pluralize a word using rules inspired by Rails' ActiveRecord Inflector.
function autoPluralize(singular: string): string {
  const lower = singular.toLowerCase()

  // Irregulars
  const irregulars: Record<string, string> = {
    person: 'people',
    child: 'children',
    man: 'men',
    woman: 'women',
    mouse: 'mice',
    goose: 'geese',
    ox: 'oxen',
    tooth: 'teeth',
    foot: 'feet',
    datum: 'data',
    index: 'indices',
  }
  if (irregulars[lower]) {
    // Preserve original casing of first letter
    const plural = irregulars[lower]
    return singular[0] === singular[0].toUpperCase()
      ? plural[0].toUpperCase() + plural.slice(1)
      : plural
  }

  // Uncountables
  const uncountables = ['fish', 'sheep', 'series', 'species', 'deer', 'moose']
  if (uncountables.includes(lower)) return singular

  // Rules (applied in order, first match wins)
  const rules: [RegExp, string][] = [
    [/(quiz)$/i, '$1zes'],
    [/(ox)$/i, '$1en'],
    [/([ml])ouse$/i, '$1ice'],
    [/(matr|vert|append)ix$/i, '$1ices'],
    [/(x|ch|ss|sh)$/i, '$1es'],
    [/([^aeiouy])y$/i, '$1ies'],
    [/(hive)$/i, '$1s'],
    [/([^f])fe$/i, '$1ves'],
    [/(lf)$/i, 'lves'],
    [/sis$/i, 'ses'],
    [/([ti])um$/i, '$1a'],
    [/(buffal|tomat|volcan)o$/i, '$1oes'],
    [/(bus)$/i, '$1es'],
    [/(alias|status)$/i, '$1es'],
    [/s$/i, 'ses'],
    [/$/, 's'],
  ]

  for (const [pattern, replacement] of rules) {
    if (pattern.test(singular)) {
      return singular.replace(pattern, replacement)
    }
  }

  return singular + 's'
}

export function pluralize(cnt: number, singular: string, plural?: string): string {
  const word = cnt === 1 ? singular : (plural ?? autoPluralize(singular))
  return `${cnt} ${word}`
}
