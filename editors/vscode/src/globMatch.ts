/**
 * globMatch.ts
 *
 * Simple glob matcher for gitignore-style patterns.
 * Ported from src/webui/frontend/src/utils/glob.ts
 */

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
function patternToRegex(pattern: string): RegExp {
  const isRooted = pattern.startsWith('/')
  if (isRooted) {
    pattern = pattern.slice(1)
  }

  const trimmedForSlashCheck = pattern.replace(/\/$/, '')
  const hasSlash = trimmedForSlashCheck.includes('/')

  let regex = ''
  let i = 0
  while (i < pattern.length) {
    const c = pattern[i]
    if (c === '*') {
      if (i + 1 < pattern.length && pattern[i + 1] === '*') {
        if (i + 2 < pattern.length && pattern[i + 2] === '/') {
          regex += '(?:.*/)?'
          i += 3
          continue
        }
        regex += '.*'
        i += 2
        continue
      }
      regex += '[^/]*'
    } else if (c === '?') {
      regex += '[^/]'
    } else if (c === '\\') {
      if (i + 1 < pattern.length) {
        i++
        regex += pattern[i].replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      }
    } else {
      regex += c.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    }
    i++
  }

  if (!isRooted && !hasSlash) {
    return new RegExp('^(?:.*/)?'+ regex + '$')
  }
  return new RegExp('^' + regex + '$')
}

export function matchesAnyPattern(path: string, patterns: string[]): boolean {
  for (const pattern of patterns) {
    if (!pattern) continue
    if (patternToRegex(pattern).test(path)) {
      return true
    }
  }
  return false
}
