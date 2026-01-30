import { useState } from 'react'
import { criticClient } from '../api/client'

export interface CommentLineInfo {
  oldFile: string
  newFile: string
  oldLine: number
  newLine: number
}

interface CommentEditorProps {
  lineInfo: CommentLineInfo
  onClose: () => void
  onSaved: () => void
}

function CommentEditor({ lineInfo, onClose, onSaved }: CommentEditorProps) {
  const [comment, setComment] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSave = async () => {
    if (!comment.trim()) {
      return
    }

    setSaving(true)
    setError(null)

    try {
      const response = await criticClient.createComment({
        oldFile: lineInfo.oldFile,
        oldLine: lineInfo.oldLine,
        newFile: lineInfo.newFile,
        newLine: lineInfo.newLine,
        comment: comment.trim(),
      })

      if (response.success) {
        onSaved()
        onClose()
      } else if (response.error) {
        setError(response.error.message || 'Failed to save comment')
      }
    } catch (err) {
      console.error('Failed to save comment:', err)
      setError(err instanceof Error ? err.message : 'Failed to save comment')
    } finally {
      setSaving(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose()
    } else if (e.key === 's' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      handleSave()
    }
  }

  const displayLine = lineInfo.newLine > 0 ? lineInfo.newLine : lineInfo.oldLine
  const displayFile = lineInfo.newFile || lineInfo.oldFile

  return (
    <div className="comment-editor-overlay" onClick={onClose}>
      <div className="comment-editor" onClick={(e) => e.stopPropagation()}>
        <div className="comment-editor-header">
          <span className="comment-editor-title">
            Add Comment - Line {displayLine}
          </span>
          <span className="comment-editor-file">{displayFile}</span>
          <button className="comment-editor-close" onClick={onClose}>
            &times;
          </button>
        </div>

        <div className="comment-editor-body">
          <textarea
            className="comment-editor-textarea"
            placeholder="Enter your comment (markdown supported)..."
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            onKeyDown={handleKeyDown}
            autoFocus
            disabled={saving}
          />
          {error && <div className="comment-editor-error">{error}</div>}
        </div>

        <div className="comment-editor-footer">
          <span className="comment-editor-hint">Ctrl+S to save, Esc to cancel</span>
          <div className="comment-editor-actions">
            <button
              className="comment-editor-button comment-editor-button-cancel"
              onClick={onClose}
              disabled={saving}
            >
              Cancel
            </button>
            <button
              className="comment-editor-button comment-editor-button-save"
              onClick={handleSave}
              disabled={saving || !comment.trim()}
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default CommentEditor
