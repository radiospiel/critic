import { useEffect, useState } from 'react'
import { criticClient } from '../api/client'
import { FileDiff, FileStatus } from '../gen/critic_pb'

interface FileListProps {
  selectedFile: string | null
  onSelectFile: (file: string, fileDiff: FileDiff) => void
}

function getFilePath(file: FileDiff): string {
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

function FileList({ selectedFile, onSelectFile }: FileListProps) {
  const [files, setFiles] = useState<FileDiff[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    criticClient
      .getDiffs({})
      .then((response) => {
        setFiles(response.diff?.files || [])
        setLoading(false)
      })
      .catch((err) => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

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
    <div>
      <div className="file-list-header">{files.length} Files</div>
      <ul className="file-list">
        {files.map((file) => {
          const path = getFilePath(file)
          return (
            <li
              key={path}
              className={`file-item ${selectedFile === path ? 'selected' : ''}`}
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
