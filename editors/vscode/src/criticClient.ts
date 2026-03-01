/**
 * criticClient.ts
 *
 * HTTP client for the critic Connect-RPC API using the JSON protocol.
 * No protobuf libraries needed — Connect-RPC supports plain JSON over HTTP.
 *
 * Endpoint format: POST <serverUrl>/critic.v1.CriticService/<MethodName>
 * Content-Type: application/json
 */

const SERVICE = 'critic.v1.CriticService'

// ---- Shared types (mirror critic.proto) ------------------------------------ //

export type ConversationStatus =
  | 'CONVERSATION_STATUS_INVALID'
  | 'CONVERSATION_STATUS_RESOLVED'
  | 'CONVERSATION_STATUS_UNRESOLVED'
  | 'CONVERSATION_STATUS_ACTIVE'
  | 'CONVERSATION_STATUS_WAITING_FOR_RESPONSE'
  | 'CONVERSATION_STATUS_INFORMAL'
  | 'CONVERSATION_STATUS_ARCHIVED'

export type ConversationType =
  | 'CONVERSATION_TYPE_INVALID'
  | 'CONVERSATION_TYPE_CONVERSATION'
  | 'CONVERSATION_TYPE_EXPLANATION'

export interface CriticMessage {
  id: string
  author: string  // "human" | "ai"
  content: string
  createdAt: string
  updatedAt: string
  isUnread: boolean
}

export interface CriticConversation {
  id: string
  status: ConversationStatus
  conversationType: ConversationType
  filePath: string
  lineNumber: number
  codeVersion: string
  context: string
  messages: CriticMessage[]
  createdAt: string
  updatedAt: string
  branchName: string
}

export interface FileSummary {
  oldPath: string
  newPath: string
  status: string
  isBinary: boolean
}

export interface FileConversationSummary {
  filePath: string
  totalCount: number
  unresolvedCount: number
  resolvedCount: number
  hasUnreadAiMessages: boolean
  explanationCount: number
}

// ---- Client class ---------------------------------------------------------- //

export class CriticClient {
  constructor(private readonly serverUrl: string) {}

  private async rpc<TReq, TRes>(method: string, request: TReq): Promise<TRes> {
    const url = `${this.serverUrl}/${SERVICE}/${method}`
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: JSON.stringify(request),
    })

    if (!response.ok) {
      const text = await response.text().catch(() => response.statusText)
      throw new Error(`critic API error ${response.status}: ${text}`)
    }

    return response.json() as Promise<TRes>
  }

  /** Check server liveness by fetching the last change timestamp. */
  async getLastChange(): Promise<{ mtimeMsecs: string }> {
    return this.rpc<object, { mtimeMsecs: string }>('GetLastChange', {})
  }

  /** Get all conversations, optionally filtered by file paths and statuses. */
  async getConversations(
    paths: string[] = [],
    statuses: ConversationStatus[] = []
  ): Promise<CriticConversation[]> {
    const res = await this.rpc<object, { conversations?: CriticConversation[] }>(
      'GetConversations',
      { paths, statuses }
    )
    return res.conversations ?? []
  }

  /** Get conversation summaries (counts) per file. */
  async getConversationsSummary(): Promise<FileConversationSummary[]> {
    const res = await this.rpc<object, { summaries?: FileConversationSummary[] }>(
      'GetConversationsSummary',
      {}
    )
    return res.summaries ?? []
  }

  /** Get diff summary (file list). */
  async getDiffSummary(): Promise<{ state: string; files: FileSummary[] }> {
    const res = await this.rpc<object, { state?: string; diff?: { files?: FileSummary[] } }>(
      'GetDiffSummary',
      {}
    )
    return {
      state: res.state ?? 'UNKNOWN',
      files: res.diff?.files ?? [],
    }
  }

  /** Create a new conversation (comment) on a diff line. */
  async createConversation(params: {
    oldFile: string
    oldLine: number
    newFile: string
    newLine: number
    comment: string
  }): Promise<boolean> {
    const res = await this.rpc<object, { success?: boolean }>(
      'CreateConversation',
      {
        oldFile: params.oldFile,
        oldLine: params.oldLine,
        newFile: params.newFile,
        newLine: params.newLine,
        comment: params.comment,
        conversationType: 'CONVERSATION_TYPE_CONVERSATION',
      }
    )
    return res.success ?? false
  }

  /** Reply to an existing conversation. */
  async replyToConversation(conversationId: string, message: string): Promise<boolean> {
    const res = await this.rpc<object, { success?: boolean }>(
      'ReplyToConversation',
      { conversationId, message }
    )
    return res.success ?? false
  }

  /** Update conversation status. */
  async markConversationAs(conversationId: string, status: ConversationStatus): Promise<void> {
    await this.rpc<object, object>('MarkConversationAs', { conversationId, status })
  }

  /** Get project config (categories, etc.). */
  async getProjectConfig(): Promise<{ categories: { name: string; patterns: string[] }[] }> {
    const res = await this.rpc<object, { categories?: { name: string; patterns: string[] }[] }>(
      'GetProjectConfig',
      {}
    )
    return { categories: res.categories ?? [] }
  }

  /** Get diff bases (base refs for diffing). */
  async getDiffBases(): Promise<{ bases: string[]; currentStart: string; currentEnd: string }> {
    const res = await this.rpc<object, { bases?: string[]; currentStart?: string; currentEnd?: string }>(
      'GetDiffBases', {}
    )
    return { bases: res.bases ?? [], currentStart: res.currentStart ?? '', currentEnd: res.currentEnd ?? '' }
  }

  /** Get server config (git root, dev mode). */
  async getConfig(): Promise<{ gitRoot: string; devMode: boolean }> {
    const res = await this.rpc<object, { gitRoot?: string; devMode?: boolean }>(
      'GetConfig',
      {}
    )
    return {
      gitRoot: res.gitRoot ?? '',
      devMode: res.devMode ?? false,
    }
  }
}
