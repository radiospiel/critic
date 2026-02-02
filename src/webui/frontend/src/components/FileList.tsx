import { useEffect, useState, useCallback, useRef } from 'react'
import { criticClient, getConversationsSummary, ConversationSummary } from '../api/client'
import { FileSummary, FileStatus } from '../gen/critic_pb'

export type FilterType = 'automatic' | 'conversations' | 'files' | 'tests' | 'hidden'

interface FileListProps {
  selectedFile: string | null
  onSelectFile: (file: string, fileSummary: FileSummary) => void
  isFocused?: boolean
  onFocus?: () => void
  onFilterChange?: (filter: FilterType) => void
}

function getFilePath(file: FileSummary): string {
  return file.status === FileStatus.DELETED ? file.oldPath : file.newPath
}

function getStatusLabel(status: FileStatus): string {
  switch (status) {
    case FileStatus.NEW:
      return 'A'
    case FileStatus.DELETED:
      return 'D'
    case FileStatus.RENAMED:
      return 'R'
    case FileStatus.UNTRACKED:
      return '?'
    case FileStatus.MODIFIED:
    default:
      return 'M'
  }
}

// Convert a gitignore-style pattern to a regex
function patternToRegex(pattern: string): RegExp {
  // Remove leading slash (makes pattern relative to root)
  const isRooted = pattern.startsWith('/')
  if (isRooted) {
    pattern = pattern.slice(1)
  }

  // Escape special regex characters except * and ?
  let regex = pattern
    .replace(/[.+^${}()|[\]\\]/g, '\\$&')
    // Convert ** to match any path
    .replace(/\*\*/g, '.*')
    // Convert * to match anything except /
    .replace(/\*/g, '[^/]*')
    // Convert ? to match single char except /
    .replace(/\?/g, '[^/]')

  // If pattern doesn't contain /, it matches anywhere in the path
  if (!pattern.includes('/')) {
    regex = '(^|/)' + regex + '$'
  } else if (isRooted) {
    regex = '^' + regex + '$'
  } else {
    regex = '(^|/)' + regex + '$'
  }

  return new RegExp(regex)
}

// Check if a file path matches any of the given patterns
function matchesPattern(path: string, patterns: string[]): boolean {
  for (const pattern of patterns) {
    // Skip empty lines and comments
    if (!pattern || pattern.startsWith('#')) continue

    const regex = patternToRegex(pattern)
    if (regex.test(path)) {
      return true
    }
  }
  return false
}

// Check if a file path matches any of the ignore patterns
function isIgnored(path: string, patterns: string[]): boolean {
  return matchesPattern(path, patterns)
}

// Check if a file path matches any of the test patterns
function isTestFile(path: string, patterns: string[]): boolean {
  return matchesPattern(path, patterns)
}

function FileList({ selectedFile, onSelectFile, isFocused, onFocus, onFilterChange }: FileListProps) {
  const [files, setFiles] = useState<FileSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [ignorePatterns, setIgnorePatterns] = useState<string[]>([])
  const [testPatterns, setTestPatterns] = useState<string[]>([])
  const [filter, setFilter] = useState<FilterType>('automatic')
  const [conversationSummaries, setConversationSummaries] = useState<Map<string, ConversationSummary>>(new Map())
  const selectedItemRef = useRef<HTMLLIElement>(null)

  // Handle user clicking a filter button
  const handleUserFilterChange = (newFilter: FilterType) => {
    setFilter(newFilter)
    onFilterChange?.(newFilter)
  }

  useEffect(() => {
    // Fetch diff summary, .criticignore, .critictest, and conversation summaries in parallel
    Promise.all([
      criticClient.getDiffSummary({}),
      criticClient.getFile({ path: '.criticignore' }).catch(() => null),
      criticClient.getFile({ path: '.critictest' }).catch(() => null),
      getConversationsSummary(),
    ])
      .then(([diffResponse, ignoreFileResponse, testFileResponse, summaryResponse]) => {
        setFiles(diffResponse.diff?.files || [])
        if (ignoreFileResponse?.content) {
          const patterns = ignoreFileResponse.content
            .split('\n')
            .map((line) => line.trim())
            .filter((line) => line && !line.startsWith('#'))
          setIgnorePatterns(patterns)
        }
        if (testFileResponse?.content) {
          const patterns = testFileResponse.content
            .split('\n')
            .map((line) => line.trim())
            .filter((line) => line && !line.startsWith('#'))
          setTestPatterns(patterns)
        }
        // Build a map of file path to conversation summary for quick lookup
        const summaryMap = new Map<string, ConversationSummary>()
        for (const summary of summaryResponse.summaries) {
          summaryMap.set(summary.filePath, summary)
        }
        setConversationSummaries(summaryMap)
        setLoading(false)
      })
      .catch((err) => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  // Scroll selected item into view
  useEffect(() => {
    if (selectedItemRef.current && isFocused) {
      selectedItemRef.current.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  }, [selectedFile, isFocused])

  // Compute visible, test, and hidden files based on patterns, sorted by path
  const { regularFiles, testFiles, hiddenFiles } = (() => {
    const regular: FileSummary[] = []
    const tests: FileSummary[] = []
    const hidden: FileSummary[] = []
    for (const file of files) {
      const path = getFilePath(file)
      if (isIgnored(path, ignorePatterns)) {
        hidden.push(file)
      } else if (isTestFile(path, testPatterns)) {
        tests.push(file)
      } else {
        regular.push(file)
      }
    }
    // Sort each category by path
    const sortByPath = (a: FileSummary, b: FileSummary) =>
      getFilePath(a).localeCompare(getFilePath(b))
    return {
      regularFiles: regular.sort(sortByPath),
      testFiles: tests.sort(sortByPath),
      hiddenFiles: hidden.sort(sortByPath),
    }
  })()

  // All visible files (regular + tests) for conversations filter
  const visibleFiles = [...regularFiles, ...testFiles]

  // Files with conversations (from visible files only), sorted by path
  const filesWithConversations = visibleFiles
    .filter((file) => {
      const path = getFilePath(file)
      const summary = conversationSummaries.get(path)
      return summary && summary.totalCount > 0
    })
    .sort((a, b) => getFilePath(a).localeCompare(getFilePath(b)))

  // Total conversation count (from visible files only)
  const totalConversations = filesWithConversations.reduce((sum, file) => {
    const path = getFilePath(file)
    const summary = conversationSummaries.get(path)
    return sum + (summary?.totalCount || 0)
  }, 0)

  // Compute effective filter: automatic means 'conversations' if any exist, otherwise 'files'
  const effectiveFilter = filter === 'automatic'
    ? (filesWithConversations.length > 0 ? 'conversations' : 'files')
    : filter

  // Determine displayed files based on filter
  const displayedFiles = (() => {
    switch (effectiveFilter) {
      case 'conversations':
        return filesWithConversations
      case 'tests':
        return testFiles
      case 'hidden':
        return hiddenFiles
      case 'files':
      default:
        return regularFiles
    }
  })()

  // Keyboard navigation when focused
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isFocused || displayedFiles.length === 0) return

      // Don't handle if in input field
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return
      }

      const currentIndex = selectedFile
        ? displayedFiles.findIndex((f) => getFilePath(f) === selectedFile)
        : -1

      switch (e.key) {
        case 'ArrowUp':
        case 'k':
          e.preventDefault()
          if (currentIndex > 0) {
            const prevFile = displayedFiles[currentIndex - 1]
            onSelectFile(getFilePath(prevFile), prevFile)
          } else if (currentIndex === -1 && displayedFiles.length > 0) {
            // No selection, select last file
            const lastFile = displayedFiles[displayedFiles.length - 1]
            onSelectFile(getFilePath(lastFile), lastFile)
          }
          break
        case 'ArrowDown':
        case 'j':
          e.preventDefault()
          if (currentIndex < displayedFiles.length - 1) {
            const nextFile = displayedFiles[currentIndex + 1]
            onSelectFile(getFilePath(nextFile), nextFile)
          } else if (currentIndex === -1 && displayedFiles.length > 0) {
            // No selection, select first file
            const firstFile = displayedFiles[0]
            onSelectFile(getFilePath(firstFile), firstFile)
          }
          break
        case 'Enter':
          // Already selected, just confirm (could switch focus to diff view)
          break
      }
    },
    [isFocused, displayedFiles, selectedFile, onSelectFile]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  if (loading) {
    return (
      <div>
        <div className="file-list-header">Files (loading)</div>
        <div className="file-list-message">Loading...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div>
        <div className="file-list-header">Files (error)</div>
        <div className="file-list-message file-list-error">{error}</div>
      </div>
    )
  }

  return (
    <div className={`file-list-container${isFocused ? ' focused' : ''}`}>
      <div className="file-list-filters">
        <button
          className={`file-list-filter-btn${effectiveFilter === 'conversations' ? ' active' : ''}`}
          onClick={() => handleUserFilterChange('conversations')}
        >
          {totalConversations} Conversations
        </button>
        <button
          className={`file-list-filter-btn${effectiveFilter === 'files' ? ' active' : ''}`}
          onClick={() => handleUserFilterChange('files')}
        >
          {regularFiles.length} Files
        </button>
        <button
          className={`file-list-filter-btn${effectiveFilter === 'tests' ? ' active' : ''}`}
          onClick={() => handleUserFilterChange('tests')}
        >
          {testFiles.length} Tests
        </button>
        <button
          className={`file-list-filter-btn${filter === 'hidden' ? ' active' : ''}`}
          onClick={() => handleUserFilterChange('hidden')}
        >
          {hiddenFiles.length} Hidden
        </button>
      </div>
      <ul className="file-list">
        {displayedFiles.map((file) => {
          const path = getFilePath(file)
          const isSelected = selectedFile === path
          const summary = conversationSummaries.get(path)
          const hasConversations = summary && summary.totalCount > 0
          const hasUnresolved = summary && summary.unresolvedCount > 0
          return (
            <li
              key={path}
              ref={isSelected ? selectedItemRef : undefined}
              className={`file-item${isSelected ? ' selected' : ''}`}
              onClick={() => {
                onSelectFile(path, file)
                onFocus?.()
              }}
              title={path}
            >
              <span className={`file-status status-${getStatusLabel(file.status).toLowerCase()}`}>
                {getStatusLabel(file.status)}
              </span>
              <span className="file-path">{path}</span>
              {hasConversations && (
                <span
                  className={`conversation-icon${hasUnresolved ? ' unresolved' : ''}`}
                  title={`${summary.totalCount} conversation${summary.totalCount > 1 ? 's' : ''} (${summary.unresolvedCount} unresolved)`}
                >
                  {summary.totalCount}
                </span>
              )}
            </li>
          )
        })}
        {displayedFiles.length === 0 && (
          <li className="file-list-message">
            {filter === 'conversations'
              ? 'No files with conversations'
              : filter === 'tests'
                ? 'No test files (configure via .critictest)'
                : filter === 'hidden'
                  ? 'Files can be hidden per .criticignore'
                  : 'No files found'}
          </li>
        )}
      </ul>
    </div>
  )
}

export default FileList
