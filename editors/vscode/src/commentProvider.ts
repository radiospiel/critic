/**
 * commentProvider.ts
 *
 * Manages VS Code comment threads that reflect critic conversations.
 * Uses the vscode.comments API to show inline review threads in the editor.
 */

import * as vscode from 'vscode'
import * as path from 'path'
import { CriticClient, CriticConversation, CriticMessage } from './criticClient'

// ---- VS Code comment types ------------------------------------------------- //

/** A single comment message inside a VS Code comment thread. */
class CriticComment implements vscode.Comment {
  body: vscode.MarkdownString | string
  mode: vscode.CommentMode
  author: vscode.CommentAuthorInformation
  label?: string
  contextValue?: string

  constructor(
    public readonly message: CriticMessage,
    public readonly conversationId: string
  ) {
    this.body = new vscode.MarkdownString(message.content)
    this.mode = vscode.CommentMode.Preview
    this.author = {
      name: message.author === 'ai' ? 'AI (critic)' : 'Human',
    }
    if (message.isUnread) {
      this.label = '● unread'
    }
    this.contextValue = `criticComment-${message.author}`
  }
}

// ---- CommentProvider class ------------------------------------------------- //

export class CriticCommentProvider implements vscode.Disposable {
  private readonly controller: vscode.CommentController
  /** Map from conversation UUID to the VS Code thread. */
  private threads = new Map<string, vscode.CommentThread>()
  /** Map from file URI string to its conversations (for quick refresh). */
  private fileConversations = new Map<string, CriticConversation[]>()

  private gitRoot = ''

  constructor(
    private readonly client: CriticClient,
    private readonly showResolved: () => boolean
  ) {
    this.controller = vscode.comments.createCommentController(
      'critic-review',
      'Critic Code Review'
    )

    // Allow users to reply from within VS Code's comment UI
    this.controller.commentingRangeProvider = {
      provideCommentingRanges: (document) => {
        // Only allow comments on files inside the git root
        if (!this.isTracked(document.uri)) {
          return []
        }
        const lineCount = document.lineCount
        return [new vscode.Range(0, 0, lineCount - 1, 0)]
      },
    }

    // Handle new comments created from the VS Code comment UI
    this.controller.options = {
      prompt: 'Add a critic review comment…',
      placeHolder: 'Enter your comment (Markdown supported)',
    }
  }

  // ---- Public API ---------------------------------------------------------- //

  setGitRoot(gitRoot: string): void {
    this.gitRoot = gitRoot
  }

  /** Replace all comment threads with the given conversations. */
  applyConversations(conversations: CriticConversation[]): void {
    // Group by file
    const byFile = new Map<string, CriticConversation[]>()
    for (const conv of conversations) {
      if (!this.shouldShow(conv)) {
        continue
      }
      const list = byFile.get(conv.filePath) ?? []
      list.push(conv)
      byFile.set(conv.filePath, list)
    }

    // Remove threads for conversations that no longer exist
    const activeIds = new Set(conversations.map((c) => c.id))
    for (const [id, thread] of this.threads) {
      if (!activeIds.has(id)) {
        thread.dispose()
        this.threads.delete(id)
      }
    }

    // Create or update threads per conversation
    for (const [filePath, convs] of byFile) {
      this.fileConversations.set(filePath, convs)
      for (const conv of convs) {
        this.upsertThread(conv)
      }
    }
  }

  /** Clear all threads (e.g. on disconnect). */
  clearAll(): void {
    for (const thread of this.threads.values()) {
      thread.dispose()
    }
    this.threads.clear()
    this.fileConversations.clear()
  }

  dispose(): void {
    this.clearAll()
    this.controller.dispose()
  }

  // ---- Commands ------------------------------------------------------------ //

  /** Handle a reply submitted via the VS Code comment UI. */
  async handleReply(reply: vscode.CommentReply): Promise<void> {
    const thread = reply.thread
    const conversationId = thread.contextValue
    if (!conversationId) {
      vscode.window.showErrorMessage('Could not determine conversation ID.')
      return
    }

    const text = reply.text.trim()
    if (!text) {
      return
    }

    // Optimistically append the comment to the thread immediately
    const optimisticComment = new CriticComment(
      {
        id: `pending-${Date.now()}`,
        author: 'human',
        content: text,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        isUnread: false,
      },
      conversationId
    )
    thread.comments = [...thread.comments, optimisticComment]

    try {
      await this.client.replyToConversation(conversationId, text)
    } catch (err) {
      // Remove optimistic comment on failure
      thread.comments = thread.comments.filter((c) => c !== optimisticComment)
      vscode.window.showErrorMessage(`Critic: failed to post reply: ${err}`)
    }
    // The poll loop will refresh the thread with the real data shortly
  }

  /** Create a new conversation on the given thread (from "+" button). */
  async handleNewComment(reply: vscode.CommentReply): Promise<void> {
    const thread = reply.thread
    const text = reply.text.trim()
    if (!text) {
      thread.dispose()
      return
    }

    const filePath = this.relPath(thread.uri.fsPath)
    if (!filePath) {
      vscode.window.showErrorMessage('Critic: file is outside the git repository.')
      thread.dispose()
      return
    }

    // Use the thread's start line (1-indexed in critic)
    const lineNo = (thread.range?.start.line ?? 0) + 1

    try {
      await this.client.createConversation({
        oldFile: filePath,
        oldLine: lineNo,
        newFile: filePath,
        newLine: lineNo,
        comment: text,
      })
      // Dispose temporary thread — the poll loop will create the real one
      thread.dispose()
    } catch (err) {
      vscode.window.showErrorMessage(`Critic: failed to create comment: ${err}`)
      thread.dispose()
    }
  }

  /** Resolve the conversation associated with this thread. */
  async resolveThread(thread: vscode.CommentThread): Promise<void> {
    const id = thread.contextValue
    if (!id) return
    try {
      await this.client.markConversationAs(id, 'CONVERSATION_STATUS_RESOLVED')
    } catch (err) {
      vscode.window.showErrorMessage(`Critic: failed to resolve: ${err}`)
    }
  }

  /** Archive the conversation associated with this thread. */
  async archiveThread(thread: vscode.CommentThread): Promise<void> {
    const id = thread.contextValue
    if (!id) return
    try {
      await this.client.markConversationAs(id, 'CONVERSATION_STATUS_ARCHIVED')
    } catch (err) {
      vscode.window.showErrorMessage(`Critic: failed to archive: ${err}`)
    }
  }

  // ---- Private helpers ----------------------------------------------------- //

  private shouldShow(conv: CriticConversation): boolean {
    if (conv.status === 'CONVERSATION_STATUS_ARCHIVED') return false
    if (conv.status === 'CONVERSATION_STATUS_RESOLVED' && !this.showResolved()) return false
    return true
  }

  private isTracked(uri: vscode.Uri): boolean {
    if (!this.gitRoot) return false
    return uri.fsPath.startsWith(this.gitRoot)
  }

  private relPath(absPath: string): string | null {
    if (!this.gitRoot) return null
    const rel = path.relative(this.gitRoot, absPath)
    if (rel.startsWith('..')) return null
    return rel
  }

  private upsertThread(conv: CriticConversation): void {
    const existing = this.threads.get(conv.id)
    const comments = conv.messages.map((m) => new CriticComment(m, conv.id))

    if (existing) {
      // Update comments and state
      existing.comments = comments
      existing.state = this.threadState(conv)
      existing.label = this.threadLabel(conv)
      return
    }

    // Create new thread
    const absPath = path.join(this.gitRoot, conv.filePath)
    const uri = vscode.Uri.file(absPath)
    // VS Code ranges are 0-indexed; critic uses 1-indexed line numbers
    const line = Math.max(0, conv.lineNumber - 1)
    const range = new vscode.Range(line, 0, line, 0)

    const thread = this.controller.createCommentThread(uri, range, comments)
    thread.label = this.threadLabel(conv)
    thread.state = this.threadState(conv)
    thread.collapsibleState = vscode.CommentThreadCollapsibleState.Expanded
    thread.contextValue = conv.id  // used to look up conversation on reply/resolve

    // Enable reply via the inline reply box
    thread.canReply = true

    this.threads.set(conv.id, thread)
  }

  private threadState(conv: CriticConversation): vscode.CommentThreadState {
    return conv.status === 'CONVERSATION_STATUS_RESOLVED'
      ? vscode.CommentThreadState.Resolved
      : vscode.CommentThreadState.Unresolved
  }

  private threadLabel(conv: CriticConversation): string {
    const isExplanation = conv.conversationType === 'CONVERSATION_TYPE_EXPLANATION'
    const hasUnread = conv.messages.some((m) => m.isUnread)
    const parts: string[] = []
    if (isExplanation) parts.push('💡 Explanation')

    if (hasUnread) parts.push('● unread AI reply')
    return parts.join(' · ') || 'Critic'
  }
}
