import { useEffect, useState } from 'react'
import { criticClient } from '../api/client'
import { FileDiff, FileStatus } from '../gen/critic_pb'

interface FileInfo {
  path: string
  status: FileStatus
}

interface FileListProps {
  selectedFile: string | null
  onSelectFile: (file: string) => void
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
  const [files, setFiles] = useState<FileInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    criticClient
      .getDiffs({})
      .then((response) => {
        const fileInfos: FileInfo[] = (response.diff?.files || []).map((f) => ({
          path: getFilePath(f),
          status: f.status,
        }))
        setFiles(fileInfos)
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
        <div className="file-list-header">Files</div>
        <div style={{ padding: '16px', color: '#666' }}>Loading...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div>
        <div className="file-list-header">Files</div>
        <div style={{ padding: '16px', color: '#c00' }}>{error}</div>
      </div>
    )
  }

  return (
    <div>
      <div className="file-list-header">Files</div>
      <ul className="file-list">
        {files.map((file) => (
          <li
            key={file.path}
            className={`file-item ${selectedFile === file.path ? 'selected' : ''}`}
            onClick={() => onSelectFile(file.path)}
            title={file.path}
          >
            <span className={`file-status status-${getStatusLabel(file.status).toLowerCase()}`}>
              {getStatusLabel(file.status)}
            </span>
            <span className="file-path">{file.path}</span>
          </li>
        ))}
        {files.length === 0 && (
          <li style={{ padding: '16px', color: '#666' }}>No files found</li>
        )}
      </ul>
    </div>
  )
}

export default FileList
