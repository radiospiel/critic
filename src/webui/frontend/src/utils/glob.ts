// patternToRegex converts a glob-like pattern to a RegExp.
// Supports a subset of .gitignore pattern syntax:
//   - "*" matches anything except "/"
//   - "?" matches a single character except "/"
//   - A leading "/" anchors the match to the repository root
//   - A pattern WITHOUT a "/" matches against the basename at any depth
//   - A pattern WITH a "/" matches against the full path from root
//   - "**/" matches in all directories
//   - "/**" matches everything inside a directory
//   - "/**/" matches zero or more directories
export function patternToRegex(pattern: string): RegExp {
  // Handle leading / (anchored to root)
  const isRooted = pattern.startsWith('/')
  if (isRooted) {
    pattern = pattern.slice(1)
  }

  // Check if pattern has a slash (other than leading).
  // If so, it matches against the full path from root.
  const trimmedForSlashCheck = pattern.replace(/\/$/, '')
  const hasSlash = trimmedForSlashCheck.includes('/')

  // Convert pattern to regex string
  let regex = ''
  let i = 0
  while (i < pattern.length) {
    const c = pattern[i]
    if (c === '*') {
      if (i + 1 < pattern.length && pattern[i + 1] === '*') {
        // ** pattern
        if (i + 2 < pattern.length && pattern[i + 2] === '/') {
          // **/ - match zero or more directories
          regex += '(?:.*/)?'
          i += 3
          continue
        }
        // ** at end or without trailing / - match everything
        regex += '.*'
        i += 2
        continue
      }
      // Single * - match anything except /
      regex += '[^/]*'
    } else if (c === '?') {
      // ? - match single character except /
      regex += '[^/]'
    } else if (c === '\\') {
      // Escape next character
      if (i + 1 < pattern.length) {
        i++
        regex += pattern[i].replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      }
    } else {
      // Escape regex metacharacters
      regex += c.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    }
    i++
  }

  // Build the full regex with appropriate anchoring
  if (!isRooted && !hasSlash) {
    // No slash in pattern: match against the basename at any depth
    return new RegExp('^(?:.*/)?'+ regex + '$')
  }
  // Has slash or is rooted: match from the root
  return new RegExp('^' + regex + '$')
}

// Check if a file path matches any of the given patterns.
// Patterns starting with "!" are negations: a file matches only if it
// matches at least one positive pattern and none of the negative patterns.
export function matchesAnyPattern(path: string, patterns: string[]): boolean {
  const positive: string[] = []
  const negative: string[] = []
  for (const p of patterns) {
    if (!p) continue
    if (p.startsWith('!')) {
      negative.push(p.slice(1))
    } else {
      positive.push(p)
    }
  }

  let matched = false
  for (const p of positive) {
    if (patternToRegex(p).test(path)) {
      matched = true
      break
    }
  }
  if (!matched) return false

  for (const p of negative) {
    if (patternToRegex(p).test(path)) {
      return false
    }
  }
  return true
}
