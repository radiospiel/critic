import { createConnectTransport } from '@connectrpc/connect-web'
import { createPromiseClient } from '@connectrpc/connect'
import { CriticService } from '../gen/critic_connect'
import { Conversation, Message, FileConversationSummary } from '../gen/critic_pb'

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

// Fetch conversations for a file using the GRPC endpoint
export async function getConversations(filePath: string): Promise<GetCommentsResult> {
  try {
    const response = await criticClient.getConversations({ path: filePath })
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

// Types for conversation summaries
export interface ConversationSummary {
  filePath: string
  totalCount: number
  unresolvedCount: number
  resolvedCount: number
  hasUnreadAiMessages: boolean
}

export interface GetConversationsSummaryResult {
  summaries: ConversationSummary[]
  error?: string
}

// Convert generated protobuf FileConversationSummary to ConversationSummary interface
function convertSummary(summary: FileConversationSummary): ConversationSummary {
  return {
    filePath: summary.filePath,
    totalCount: summary.totalCount,
    unresolvedCount: summary.unresolvedCount,
    resolvedCount: summary.resolvedCount,
    hasUnreadAiMessages: summary.hasUnreadAiMessages,
  }
}

// Fetch conversation summaries for all files
export async function getConversationsSummary(): Promise<GetConversationsSummaryResult> {
  try {
    const response = await criticClient.getConversationsSummary({})
    if (response.error) {
      return {
        summaries: [],
        error: response.error.message,
      }
    }
    return {
      summaries: response.summaries.map(convertSummary),
    }
  } catch (err) {
    return {
      summaries: [],
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

// Types for reply to conversation
export interface ReplyToConversationResult {
  success: boolean
  error?: string
}

// Reply to an existing conversation
export async function replyToConversation(conversationId: string, message: string): Promise<ReplyToConversationResult> {
  try {
    const response = await criticClient.replyToConversation({ conversationId, message })
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
