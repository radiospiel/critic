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

// Alias for new naming convention
export type GetConversationsResult = GetCommentsResult

// Types for conversation summary
export interface FileConversationSummary {
  filePath: string
  totalCount: number
  unresolvedCount: number
  resolvedCount: number
  hasUnreadAiMessages: boolean
}

export interface GetConversationsSummaryResult {
  summaries: FileConversationSummary[]
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

// Types for diff bases
export interface DiffBasesResult {
  bases: string[]
  currentBase: string
  error?: string
}

export interface SetDiffBaseResult {
  success: boolean
  error?: string
}

// Fetch available diff bases and current selection
export async function getDiffBases(): Promise<DiffBasesResult> {
  try {
    const response = await criticClient.getDiffBases({})
    if (response.error) {
      return {
        bases: [],
        currentBase: '',
        error: response.error.message,
      }
    }
    return {
      bases: response.bases,
      currentBase: response.currentBase,
    }
  } catch (err) {
    return {
      bases: [],
      currentBase: '',
      error: err instanceof Error ? err.message : 'Unknown error',
    }
  }
}

// Set the current diff base
export async function setDiffBase(base: string): Promise<SetDiffBaseResult> {
  try {
    const response = await criticClient.setDiffBase({ base })
    if (response.error) {
      return {
        success: false,
        error: response.error.message,
      }
    }
    return {
      success: response.success,
    }
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : 'Unknown error',
    }
  }
}

// Alias for new naming convention (GetConversations = GetComments)
export const getConversations = getComments

// Fetch conversation summaries for all files
export async function getConversationsSummary(): Promise<GetConversationsSummaryResult> {
  try {
    // Use the REST endpoint since proto isn't regenerated yet
    const response = await fetch('/api/conversations/summary')
    if (!response.ok) {
      return {
        summaries: [],
        error: `HTTP error: ${response.status}`,
      }
    }
    const data = await response.json()
    if (data.error) {
      return {
        summaries: [],
        error: data.error.message,
      }
    }
    // Convert snake_case from JSON to camelCase
    const summaries: FileConversationSummary[] = (data.summaries || []).map(
      (s: {
        file_path: string
        total_count: number
        unresolved_count: number
        resolved_count: number
        has_unread_ai_messages: boolean
      }) => ({
        filePath: s.file_path,
        totalCount: s.total_count,
        unresolvedCount: s.unresolved_count,
        resolvedCount: s.resolved_count,
        hasUnreadAiMessages: s.has_unread_ai_messages,
      })
    )
    return { summaries }
  } catch (err) {
    return {
      summaries: [],
      error: err instanceof Error ? err.message : 'Unknown error',
    }
  }
}
