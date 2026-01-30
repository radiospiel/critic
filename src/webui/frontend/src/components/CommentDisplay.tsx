import { CommentConversation, CommentMessage } from '../api/client'

interface CommentDisplayProps {
  conversations: CommentConversation[]
  lineNumber: number
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
      <div className="comment-message-content">{message.content}</div>
    </div>
  )
}

function ConversationItem({ conversation }: { conversation: CommentConversation }) {
  return (
    <div className={`comment-conversation comment-conversation-${conversation.status}`}>
      <div className="comment-conversation-header">
        <span className="comment-conversation-line">Line {conversation.lineNumber}</span>
        <span className={`comment-conversation-status comment-status-${conversation.status}`}>
          {conversation.status}
        </span>
      </div>
      <div className="comment-conversation-messages">
        {conversation.messages.map((message) => (
          <MessageItem key={message.id} message={message} />
        ))}
      </div>
    </div>
  )
}

function CommentDisplay({ conversations, lineNumber }: CommentDisplayProps) {
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
        <ConversationItem key={conversation.id} conversation={conversation} />
      ))}
    </div>
  )
}

export default CommentDisplay
