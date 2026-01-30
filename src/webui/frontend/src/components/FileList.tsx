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

function FileList({ selectedFile, onSelectFile, isFocused }: FileListProps) {
  const [files, setFiles] = useState<FileSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const selectedItemRef = useRef<HTMLLIElement>(null)

  useEffect(() => {
    criticClient
      .getDiffSummary({})
      .then((response) => {
        setFiles(response.diff?.files || [])
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

  // Keyboard navigation when focused
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isFocused || files.length === 0) return

      // Don't handle if in input field
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return
      }

      const currentIndex = selectedFile
        ? files.findIndex((f) => getFilePath(f) === selectedFile)
        : -1

      switch (e.key) {
        case 'ArrowUp':
        case 'k':
          e.preventDefault()
          if (currentIndex > 0) {
            const prevFile = files[currentIndex - 1]
            onSelectFile(getFilePath(prevFile), prevFile)
          } else if (currentIndex === -1 && files.length > 0) {
            // No selection, select last file
            const lastFile = files[files.length - 1]
            onSelectFile(getFilePath(lastFile), lastFile)
          }
          break
        case 'ArrowDown':
        case 'j':
          e.preventDefault()
          if (currentIndex < files.length - 1) {
            const nextFile = files[currentIndex + 1]
            onSelectFile(getFilePath(nextFile), nextFile)
          } else if (currentIndex === -1 && files.length > 0) {
            // No selection, select first file
            const firstFile = files[0]
            onSelectFile(getFilePath(firstFile), firstFile)
          }
          break
        case 'Enter':
          // Already selected, just confirm (could switch focus to diff view)
          break
      }
    },
    [isFocused, files, selectedFile, onSelectFile]
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
      <div className="file-list-header">{files.length} Files</div>
      <ul className="file-list">
        {files.map((file) => {
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
        {files.length === 0 && (
          <li className="file-list-message">No files found</li>
        )}
      </ul>
    </div>
  )
}

export default FileList
