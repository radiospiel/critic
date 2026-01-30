import { useEffect, useState, useCallback, useRef } from 'react'
import { criticClient } from '../api/client'
import { FileSummary, FileStatus } from '../gen/critic_pb'

interface FileListProps {
  selectedFile: string | null
  onSelectFile: (file: string, fileSummary: FileSummary) => void
  isFocused?: boolean
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

// Check if a file path matches any of the ignore patterns
function isIgnored(path: string, patterns: string[]): boolean {
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

function FileList({ selectedFile, onSelectFile, isFocused }: FileListProps) {
  const [files, setFiles] = useState<FileSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [ignorePatterns, setIgnorePatterns] = useState<string[]>([])
  const [showHiddenOnly, setShowHiddenOnly] = useState(false)
  const selectedItemRef = useRef<HTMLLIElement>(null)

  useEffect(() => {
    // Fetch both diff summary and .criticignore in parallel
    Promise.all([
      criticClient.getDiffSummary({}),
      criticClient.getFile({ path: '.criticignore' }).catch(() => null),
    ])
      .then(([diffResponse, fileResponse]) => {
        setFiles(diffResponse.diff?.files || [])
        if (fileResponse?.content) {
          const patterns = fileResponse.content
            .split('\n')
            .map((line) => line.trim())
            .filter((line) => line && !line.startsWith('#'))
          setIgnorePatterns(patterns)
        }
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

  // Compute visible and hidden files based on ignore patterns
  const { visibleFiles, hiddenFiles } = (() => {
    if (ignorePatterns.length === 0) {
      return { visibleFiles: files, hiddenFiles: [] as FileSummary[] }
    }
    const visible: FileSummary[] = []
    const hidden: FileSummary[] = []
    for (const file of files) {
      const path = getFilePath(file)
      if (isIgnored(path, ignorePatterns)) {
        hidden.push(file)
      } else {
        visible.push(file)
      }
    }
    return { visibleFiles: visible, hiddenFiles: hidden }
  })()

  // Files to display based on showHiddenOnly toggle
  const displayedFiles = showHiddenOnly ? hiddenFiles : visibleFiles

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

  const headerText = showHiddenOnly
    ? `${hiddenFiles.length} Hidden Files`
    : `${visibleFiles.length} Files`

  return (
    <div className={`file-list-container${isFocused ? ' focused' : ''}`}>
      <div className="file-list-header">
        {headerText}
        {hiddenFiles.length > 0 && !showHiddenOnly && (
          <span
            className="hidden-files-toggle"
            onClick={() => setShowHiddenOnly(true)}
            title="Click to show hidden files"
          >
            {' '}(+ {hiddenFiles.length} hidden)
          </span>
        )}
        {showHiddenOnly && (
          <span
            className="hidden-files-toggle"
            onClick={() => setShowHiddenOnly(false)}
            title="Click to show visible files"
          >
            {' '}(back to visible)
          </span>
        )}
      </div>
      <ul className="file-list">
        {displayedFiles.map((file) => {
          const path = getFilePath(file)
          const isSelected = selectedFile === path
          return (
            <li
              key={path}
              ref={isSelected ? selectedItemRef : undefined}
              className={`file-item${isSelected ? ' selected' : ''}`}
              onClick={() => onSelectFile(path, file)}
              title={path}
            >
              <span className={`file-status status-${getStatusLabel(file.status).toLowerCase()}`}>
                {getStatusLabel(file.status)}
              </span>
              <span className="file-path">{path}</span>
            </li>
          )
        })}
        {displayedFiles.length === 0 && (
          <li className="file-list-message">
            {showHiddenOnly ? 'No hidden files' : 'No files found'}
          </li>
        )}
      </ul>
    </div>
  )
}

export default FileList
