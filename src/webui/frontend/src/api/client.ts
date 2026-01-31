import { createConnectTransport } from '@connectrpc/connect-web'
import { createPromiseClient } from '@connectrpc/connect'
import { CriticService } from '../gen/critic_connect'
import { Conversation, Message } from '../gen/critic_pb'

// Create a transport for the Connect protocol
const transport = createConnectTransport({
  baseUrl: window.location.origin,
})

// Create the client
export const criticClient = createPromiseClient(CriticService, transport)

// Types for comments (for component compatibility)
export interface CommentMessage {
  id: string
  author: string
  content: string
  createdAt: string
  updatedAt: string
  isUnread: boolean
}

export interface CommentConversation {
  id: string
  status: string
  filePath: string
  lineNumber: number
  codeVersion: string
  context: string
  messages: CommentMessage[]
  createdAt: string
  updatedAt: string
}

export interface GetCommentsResult {
  conversations: CommentConversation[]
  error?: string
}

// Convert generated protobuf Message to CommentMessage interface
function convertMessage(msg: Message): CommentMessage {
  return {
    id: msg.id,
    author: msg.author,
    content: msg.content,
    createdAt: msg.createdAt,
    updatedAt: msg.updatedAt,
    isUnread: msg.isUnread,
  }
}

// Convert generated protobuf Conversation to CommentConversation interface
function convertConversation(conv: Conversation): CommentConversation {
  return {
    id: conv.id,
    status: conv.status,
    filePath: conv.filePath,
    lineNumber: conv.lineNumber,
    codeVersion: conv.codeVersion,
    context: conv.context,
    messages: conv.messages.map(convertMessage),
    createdAt: conv.createdAt,
    updatedAt: conv.updatedAt,
  }
}

// Fetch comments for a file using the GRPC endpoint
export async function getComments(filePath: string): Promise<GetCommentsResult> {
  try {
    const response = await criticClient.getComments({ path: filePath })
    if (response.error) {
      return {
        conversations: [],
        error: response.error.message,
      }
    }
    return {
      conversations: response.conversations.map(convertConversation),
    }
  } catch (err) {
    return {
      conversations: [],
      error: err instanceof Error ? err.message : 'Unknown error',
    }
  }
}
