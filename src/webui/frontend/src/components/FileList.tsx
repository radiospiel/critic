import { useEffect, useState } from 'react'

interface FileListProps {
  selectedFile: string | null
  onSelectFile: (file: string) => void
}

function FileList({ selectedFile, onSelectFile }: FileListProps) {
  const [files, setFiles] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/files')
      .then((res) => {
        if (!res.ok) throw new Error('Failed to fetch files')
        return res.json()
      })
      .then((data) => {
        setFiles(data.files || [])
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
            key={file}
            className={`file-item ${selectedFile === file ? 'selected' : ''}`}
            onClick={() => onSelectFile(file)}
            title={file}
          >
            {file}
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
