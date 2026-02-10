import { useState, useRef, useEffect } from 'react'
import Markdown from 'react-markdown'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import { CommentConversation, CommentMessage, replyToConversation, resolveConversation } from '../api/client'

interface CommentDisplayProps {
  conversations: CommentConversation[]
  lineNumber: number
  onReplyAdded?: () => void
  scrollToBottom?: boolean
  alwaysShowEditor?: boolean
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleString()
}

function MessageItem({ message }: { message: CommentMessage }) {
  return (
    <div className={`comment-message comment-message-${message.author}`}>
      <div className="comment-message-header">
        <span className="comment-message-author">{message.author === 'ai' ? 'Bot' : message.author}</span>
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
  alwaysShowEditor?: boolean
}

function ConversationItem({ conversation, onReplyAdded, alwaysShowEditor }: ConversationItemProps) {
  const [showEditor, setShowEditor] = useState(!!alwaysShowEditor)
  const [resolving, setResolving] = useState(false)

  const handleReplySaved = () => {
    if (!alwaysShowEditor) setShowEditor(false)
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

  const isExplanation = conversation.conversationType === 'explanation'

  return (
    <div className={`comment-conversation comment-conversation-${conversation.status}${isExplanation ? ' conversation-type-explanation' : ''}`}>
      <div className="comment-conversation-header">
        {isExplanation && (
          <svg className="explanation-icon" width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 1.5c-2.363 0-4 1.69-4 3.75 0 .984.424 1.625.984 2.304l.214.253c.223.264.47.556.673.848.284.411.537.896.621 1.49a.75.75 0 0 1-1.484.211c-.04-.282-.163-.547-.37-.847a8.456 8.456 0 0 0-.542-.68c-.084-.1-.173-.205-.268-.32C3.201 7.75 2.5 6.766 2.5 5.25 2.5 2.31 4.863.5 8 .5s5.5 1.81 5.5 4.75c0 1.516-.701 2.5-1.328 3.259-.095.115-.184.22-.268.319-.207.245-.383.453-.541.681-.208.3-.33.565-.37.847a.751.751 0 0 1-1.485-.212c.084-.593.337-1.078.621-1.489.203-.292.45-.584.673-.848.075-.088.147-.173.213-.253.561-.679.985-1.32.985-2.304 0-2.06-1.637-3.75-4-3.75ZM5.75 12h4.5a.75.75 0 0 1 0 1.5h-4.5a.75.75 0 0 1 0-1.5ZM6 15.25a.75.75 0 0 1 .75-.75h2.5a.75.75 0 0 1 0 1.5h-2.5a.75.75 0 0 1-.75-.75Z"/>
          </svg>
        )}
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
          onCancel={() => { if (!alwaysShowEditor) setShowEditor(false) }}
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

function CommentDisplay({ conversations, lineNumber, onReplyAdded, scrollToBottom, alwaysShowEditor }: CommentDisplayProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  // Filter conversations for this specific line
  const lineConversations = conversations.filter(
    (conv) => conv.lineNumber === lineNumber
  )

  const messageCount = lineConversations.reduce((sum, c) => sum + c.messages.length, 0)

  useEffect(() => {
    if (!scrollToBottom) return
    // Delay to allow async editors (TipTap) to fully render
    const timer = setTimeout(() => {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, 100)
    return () => clearTimeout(timer)
  }, [scrollToBottom, messageCount])

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
          alwaysShowEditor={alwaysShowEditor}
        />
      ))}
      <div ref={bottomRef} />
    </div>
  )
}

export default CommentDisplay
