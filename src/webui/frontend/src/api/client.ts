import { createConnectTransport } from '@connectrpc/connect-web'
import { createPromiseClient } from '@connectrpc/connect'
import { CriticService } from '../gen/critic_connect'
import { Conversation, Message, FileConversationSummary, ConversationStatus, ConversationType } from '../gen/critic_pb'

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
  conversationType: string
  filePath: string
  lineNumber: number
  codeVersion: string
  context: string
  messages: CommentMessage[]
  createdAt: string
  updatedAt: string
  branchName: string
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

// Convert ConversationStatus enum to string
function statusToString(status: ConversationStatus): string {
  switch (status) {
    case ConversationStatus.RESOLVED:
      return 'resolved'
    case ConversationStatus.UNRESOLVED:
      return 'unresolved'
    case ConversationStatus.ACTIVE:
      return 'active'
    case ConversationStatus.WAITING_FOR_RESPONSE:
      return 'waiting_for_response'
    case ConversationStatus.INFORMAL:
      return 'informal'
    case ConversationStatus.ARCHIVED:
      return 'archived'
    default:
      return 'invalid'
  }
}

// Convert ConversationType enum to string
function conversationTypeToString(ct: ConversationType): string {
  switch (ct) {
    case ConversationType.EXPLANATION:
      return 'explanation'
    case ConversationType.CONVERSATION:
      return 'conversation'
    default:
      return 'conversation'
  }
}

// Convert generated protobuf Conversation to CommentConversation interface
function convertConversation(conv: Conversation): CommentConversation {
  return {
    id: conv.id,
    status: statusToString(conv.status),
    conversationType: conversationTypeToString(conv.conversationType),
    filePath: conv.filePath,
    lineNumber: conv.lineNumber,
    codeVersion: conv.codeVersion,
    context: conv.context,
    messages: conv.messages.map(convertMessage),
    createdAt: conv.createdAt,
    updatedAt: conv.updatedAt,
    branchName: conv.branchName,
  }
}

// Fetch conversations matching the given filters.
// If paths is empty, returns conversations across all files.
// If statuses is empty, returns conversations with any status.
export async function getConversations(paths?: string[], statuses?: ConversationStatus[]): Promise<GetCommentsResult> {
  try {
    const response = await criticClient.getConversations({
      paths: paths || [],
      statuses: statuses || [],
    })
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
  explanationCount: number
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
    explanationCount: summary.explanationCount,
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
  currentStart: string
  currentEnd: string
  error?: string
}

export interface SetDiffBaseResult {
  success: boolean
  error?: string
}

// Fetch available diff bases and current range selection
export async function getDiffBases(): Promise<DiffBasesResult> {
  try {
    const response = await criticClient.getDiffBases({})
    if (response.error) {
      return {
        bases: [],
        currentStart: '',
        currentEnd: '',
        error: response.error.message,
      }
    }
    return {
      bases: response.bases,
      currentStart: response.currentStart,
      currentEnd: response.currentEnd,
    }
  } catch (err) {
    return {
      bases: [],
      currentStart: '',
      currentEnd: '',
      error: err instanceof Error ? err.message : 'Unknown error',
    }
  }
}

// Set the current diff range (start and optional end)
export async function setDiffRange(start: string, end: string): Promise<SetDiffBaseResult> {
  try {
    const response = await criticClient.setDiffBase({ start, end })
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

// Types for conversation status updates
export interface MarkConversationResult {
  success: boolean
  error?: string
}

// Update conversation status via the unified MarkConversationAs RPC
async function markConversationAs(conversationId: string, status: ConversationStatus): Promise<MarkConversationResult> {
  try {
    const response = await criticClient.markConversationAs({ conversationId, status })
    if (response.error) {
      return {
        success: false,
        error: response.error.message,
      }
    }
    return {
      success: true,
    }
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : 'Unknown error',
    }
  }
}

// Resolve a conversation
export async function resolveConversation(conversationId: string): Promise<MarkConversationResult> {
  return markConversationAs(conversationId, ConversationStatus.RESOLVED)
}

// Archive a conversation
export async function archiveConversation(conversationId: string): Promise<MarkConversationResult> {
  return markConversationAs(conversationId, ConversationStatus.ARCHIVED)
}

// Unresolve a conversation
export async function unresolveConversation(conversationId: string): Promise<MarkConversationResult> {
  return markConversationAs(conversationId, ConversationStatus.UNRESOLVED)
}

// Types for root conversation
export interface GetRootConversationResult {
  conversation: CommentConversation | null
  error?: string
}

// Fetch the root conversation (global announcements)
export async function getRootConversation(): Promise<GetRootConversationResult> {
  try {
    const response = await criticClient.getRootConversation({})
    if (response.error) {
      return {
        conversation: null,
        error: response.error.message,
      }
    }
    return {
      conversation: response.conversation ? convertConversation(response.conversation) : null,
    }
  } catch (err) {
    return {
      conversation: null,
      error: err instanceof Error ? err.message : 'Unknown error',
    }
  }
}

// Types for server config
export interface ServerConfig {
  gitRoot: string
  devMode: boolean
  editorUrl: string
}

export interface GetConfigResult {
  config: ServerConfig | null
  error?: string
}

// Fetch server configuration (merges server config + project config editor URL)
export async function getConfig(): Promise<GetConfigResult> {
  try {
    const [configResponse, projectResponse] = await Promise.all([
      criticClient.getConfig({}),
      criticClient.getProjectConfig({}).catch(() => null),
    ])
    if (configResponse.error) {
      return {
        config: null,
        error: configResponse.error.message,
      }
    }
    return {
      config: {
        gitRoot: configResponse.gitRoot,
        devMode: configResponse.devMode,
        editorUrl: projectResponse?.editor?.url || '',
      },
    }
  } catch (err) {
    return {
      config: null,
      error: err instanceof Error ? err.message : 'Unknown error',
    }
  }
}
