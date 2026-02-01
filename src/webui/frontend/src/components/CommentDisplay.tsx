import { useState } from 'react'
import Markdown from 'react-markdown'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import { CommentConversation, CommentMessage, replyToConversation, resolveConversation } from '../api/client'

interface CommentDisplayProps {
  conversations: CommentConversation[]
  lineNumber: number
  onReplyAdded?: () => void
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleString()
}

function MessageItem({ message }: { message: CommentMessage }) {
  return (
    <div className={`comment-message comment-message-${message.author}`}>
      <div className="comment-message-header">
        <span className="comment-message-author">{message.author}</span>
        <span className="comment-message-time">{formatDate(message.createdAt)}</span>
        {message.isUnread && <span className="comment-message-unread">New</span>}
      </div>
      <div className="comment-message-content">
        <Markdown>{message.content}</Markdown>
      </div>
    </div>
  )
}

interface ReplyEditorProps {
  conversationId: string
  onReplySaved: () => void
  onCancel: () => void
}

function ReplyEditor({ conversationId, onReplySaved, onCancel }: ReplyEditorProps) {
  const [saving, setSaving] = useState(false)

  const editor = useEditor({
    extensions: [
      StarterKit,
      Placeholder.configure({
        placeholder: 'Write a reply...',
      }),
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'reply-editor-content',
        autocorrect: 'off',
        autocapitalize: 'off',
        spellcheck: 'false',
      },
    },
    autofocus: true,
  })

  const handleSave = async () => {
    if (!editor || saving) return
    const content = editor.getText().trim()
    if (!content) return

    setSaving(true)
    try {
      const result = await replyToConversation(conversationId, content)
      if (result.success) {
        editor.commands.clearContent()
        onReplySaved()
      } else {
        console.error('Failed to save reply:', result.error)
      }
    } finally {
      setSaving(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && e.metaKey) {
      e.preventDefault()
      handleSave()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      onCancel()
    }
  }

  return (
    <div className="reply-editor" onKeyDown={handleKeyDown}>
      <EditorContent editor={editor} />
      <div className="reply-editor-actions">
        <span className="reply-editor-hint"><kbd>⌘</kbd> + <kbd>↵</kbd> to save, <kbd>Esc</kbd> to cancel</span>
        <button
          className="reply-editor-button"
          onClick={handleSave}
          disabled={saving}
        >
          {saving ? 'Saving...' : 'Save'}
        </button>
      </div>
    </div>
  )
}

interface ConversationItemProps {
  conversation: CommentConversation
  onReplyAdded?: () => void
}

function ConversationItem({ conversation, onReplyAdded }: ConversationItemProps) {
  const [showEditor, setShowEditor] = useState(false)
  const [resolving, setResolving] = useState(false)

  const handleReplySaved = () => {
    setShowEditor(false)
    onReplyAdded?.()
  }

  const handleResolve = async () => {
    setResolving(true)
    try {
      const result = await resolveConversation(conversation.id)
      if (result.success) {
        onReplyAdded?.()
      } else {
        console.error('Failed to resolve conversation:', result.error)
      }
    } finally {
      setResolving(false)
    }
  }

  return (
    <div className={`comment-conversation comment-conversation-${conversation.status}`}>
      <div className="comment-conversation-header">
        <span className={`comment-conversation-status comment-status-${conversation.status}`}>
          {conversation.status}
        </span>
      </div>
      <div className="comment-conversation-messages">
        {conversation.messages.map((message) => (
          <MessageItem key={message.id} message={message} />
        ))}
      </div>
      {showEditor ? (
        <ReplyEditor
          conversationId={conversation.id}
          onReplySaved={handleReplySaved}
          onCancel={() => setShowEditor(false)}
        />
      ) : (
        <div className="reply-button-container">
          <button className="reply-button" onClick={() => setShowEditor(true)}>
            Reply
          </button>
          {conversation.status === 'unresolved' && (
            <button
              className="resolve-button"
              onClick={handleResolve}
              disabled={resolving}
            >
              {resolving ? 'Resolving...' : 'Resolve'}
            </button>
          )}
        </div>
      )}
    </div>
  )
}

function CommentDisplay({ conversations, lineNumber, onReplyAdded }: CommentDisplayProps) {
  // Filter conversations for this specific line
  const lineConversations = conversations.filter(
    (conv) => conv.lineNumber === lineNumber
  )

  if (lineConversations.length === 0) {
    return null
  }

  return (
    <div className="comment-display">
      {lineConversations.map((conversation) => (
        <ConversationItem
          key={conversation.id}
          conversation={conversation}
          onReplyAdded={onReplyAdded}
        />
      ))}
    </div>
  )
}

export default CommentDisplay
