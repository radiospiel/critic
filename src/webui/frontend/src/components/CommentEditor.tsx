import { useEffect } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Typography from '@tiptap/extension-typography'
import { criticClient } from '../api/client'

export interface CommentLineInfo {
  oldFile: string
  newFile: string
  oldLine: number
  newLine: number
}

interface InlineCommentEditorProps {
  lineInfo: CommentLineInfo
  onClose: () => void
  onSaved: () => void
}

function InlineCommentEditor({ lineInfo, onClose, onSaved }: InlineCommentEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: {
          levels: [1, 2, 3],
        },
      }),
      Placeholder.configure({
        placeholder: 'Write a comment... (Markdown shortcuts supported)',
      }),
      Typography,
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'inline-comment-editor-content',
      },
    },
  })

  useEffect(() => {
    if (editor) {
      editor.commands.focus()
    }
  }, [editor])

  const handleSave = async () => {
    if (!editor) return

    const content = editor.getText()
    if (!content.trim()) {
      return
    }

    try {
      const response = await criticClient.createComment({
        oldFile: lineInfo.oldFile,
        oldLine: lineInfo.oldLine,
        newFile: lineInfo.newFile,
        newLine: lineInfo.newLine,
        comment: content.trim(),
      })

      if (response.success) {
        onSaved()
        onClose()
      } else if (response.error) {
        console.error('Failed to save comment:', response.error.message)
      }
    } catch (err) {
      console.error('Failed to save comment:', err)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      onClose()
    } else if (e.key === 's' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      handleSave()
    }
  }

  return (
    <div className="inline-comment-editor" onKeyDown={handleKeyDown}>
      <EditorContent editor={editor} />
      <div className="inline-comment-editor-actions">
        <span className="inline-comment-editor-hint">Ctrl+S to save, Esc to cancel</span>
        <button
          className="inline-comment-button inline-comment-button-cancel"
          onClick={onClose}
        >
          Cancel
        </button>
        <button
          className="inline-comment-button inline-comment-button-save"
          onClick={handleSave}
          disabled={!editor?.getText().trim()}
        >
          Comment
        </button>
      </div>
    </div>
  )
}

export default InlineCommentEditor
