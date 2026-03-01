/**
 * extension.ts
 *
 * VS Code extension for the critic code review tool.
 *
 * Lifecycle:
 *  activate() → connect to critic server → poll for conversations → render threads
 *  deactivate() → stop polling, dispose resources
 */

import * as vscode from 'vscode'
import { CriticClient } from './criticClient'
import { CriticCommentProvider } from './commentProvider'
import { FileListProvider, CategoryNode } from './fileListProvider'
import { BaseFileProvider, makeBaseUri } from './baseFileProvider'

// ---- Globals --------------------------------------------------------------- //

let pollTimer: ReturnType<typeof setInterval> | undefined
let statusBarItem: vscode.StatusBarItem | undefined
let commentProvider: CriticCommentProvider | undefined
let fileListProvider: FileListProvider | undefined
let baseFileProvider: BaseFileProvider | undefined
let client: CriticClient | undefined

/** Track the last known mtime to avoid unnecessary refreshes. */
let lastMtime = ''
/** Track whether we are currently connected. */
let connected = false
/** Base ref for diff views (e.g. a commit hash). */
let currentStart = ''

// ---- Activation ------------------------------------------------------------ //

export function activate(context: vscode.ExtensionContext): void {
  // Status bar item (bottom bar)
  statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100)
  statusBarItem.command = 'critic.statusBarMenu'
  statusBarItem.text = '$(comment-discussion) Critic'
  statusBarItem.tooltip = 'Critic Code Review — click to open Web UI'
  statusBarItem.show()
  context.subscriptions.push(statusBarItem)

  // Reply handler — registered once, delegates to current commentProvider
  context.subscriptions.push(
    vscode.commands.registerCommand(
      '_critic.internal.reply',
      async (reply: vscode.CommentReply) => {
        if (reply.thread.contextValue) {
          await commentProvider?.handleReply(reply)
        } else {
          await commentProvider?.handleNewComment(reply)
        }
        // Poll after the API call completes, plus a short delay for server processing
        setTimeout(() => void poll(true), 500)
      }
    )
  )

  // Re-init when configuration changes
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (
        e.affectsConfiguration('critic.serverUrl') ||
        e.affectsConfiguration('critic.pollIntervalMs') ||
        e.affectsConfiguration('critic.showResolvedComments')
      ) {
        restartPolling()
      }
    })
  )

  // Commands
  context.subscriptions.push(
    vscode.commands.registerCommand('critic.refresh', () => {
      void poll(true)
    }),

    vscode.commands.registerCommand('critic.openWebUI', () => {
      const url = getConfig().serverUrl
      void vscode.env.openExternal(vscode.Uri.parse(url))
    }),

    vscode.commands.registerCommand(
      'critic.resolveConversation',
      (thread: vscode.CommentThread) => {
        void commentProvider?.resolveThread(thread)
        // Refresh soon
        setTimeout(() => void poll(true), 500)
      }
    ),

    vscode.commands.registerCommand(
      'critic.archiveConversation',
      (thread: vscode.CommentThread) => {
        void commentProvider?.archiveThread(thread)
        setTimeout(() => void poll(true), 500)
      }
    ),

    vscode.commands.registerCommand(
      'critic.openFile',
      (uri: vscode.Uri, filePath?: string, fileStatus?: string, baseRef?: string) => {
        if (!uri) return

        const status = (fileStatus ?? '').toUpperCase()
        if (baseRef && (status === 'MODIFIED' || status === 'M' || status === 'RENAMED' || status === 'R')) {
          const baseUri = makeBaseUri(filePath ?? '', baseRef)
          const title = `${filePath} (base ↔ working)`
          void vscode.commands.executeCommand('vscode.diff', baseUri, uri, title)
          return
        }

        if (baseRef && (status === 'DELETED' || status === 'D')) {
          const baseUri = makeBaseUri(filePath ?? '', baseRef)
          const title = `${filePath} (deleted)`
          void vscode.commands.executeCommand('vscode.diff', baseUri, uri, title)
          return
        }

        void vscode.window.showTextDocument(uri, { preview: false })
      }
    ),

    vscode.commands.registerCommand('critic.statusBarMenu', async () => {
      const pick = await vscode.window.showQuickPick(
        [
          { label: '$(link-external) Open Web UI in Browser', action: 'webui' },
          { label: '$(refresh) Refresh Comments', action: 'refresh' },
          { label: '$(debug-restart) VS Code Developer: Reload Window', action: 'reload' },
        ],
        { placeHolder: 'Critic' }
      )
      if (!pick) return
      switch (pick.action) {
        case 'webui':
          void vscode.commands.executeCommand('critic.openWebUI')
          break
        case 'refresh':
          void vscode.commands.executeCommand('critic.refresh')
          break
        case 'reload':
          void vscode.commands.executeCommand('workbench.action.reloadWindow')
          break
      }
    })
  )

  // Base file content provider for diff views
  baseFileProvider = new BaseFileProvider()
  context.subscriptions.push(
    vscode.workspace.registerTextDocumentContentProvider('critic-base', baseFileProvider)
  )

  // File list tree view (Activity Bar sidebar)
  fileListProvider = new FileListProvider()
  const treeView = vscode.window.createTreeView('critic.fileList', {
    treeDataProvider: fileListProvider,
    showCollapseAll: false,
  })
  treeView.onDidExpandElement((e) => {
    if (e.element instanceof CategoryNode) {
      fileListProvider?.onDidExpand(e.element)
    }
  })
  treeView.onDidCollapseElement((e) => {
    if (e.element instanceof CategoryNode) {
      fileListProvider?.onDidCollapse(e.element)
    }
  })
  context.subscriptions.push(treeView)

  // Initialize client and comment provider (after commands, so activation succeeds even if server is down)
  initClient()
}

export function deactivate(): void {
  stopPolling()
  commentProvider?.dispose()
  commentProvider = undefined
  fileListProvider?.dispose()
  fileListProvider = undefined
  statusBarItem?.dispose()
  statusBarItem = undefined
}

// ---- Setup / teardown ------------------------------------------------------ //

function getConfig(): { serverUrl: string; pollIntervalMs: number; showResolved: boolean } {
  const cfg = vscode.workspace.getConfiguration('critic')
  return {
    serverUrl: cfg.get<string>('serverUrl', 'http://localhost:65432'),
    pollIntervalMs: cfg.get<number>('pollIntervalMs', 5000),
    showResolved: cfg.get<boolean>('showResolvedComments', false),
  }
}

function initClient(): void {
  const { serverUrl, pollIntervalMs, showResolved } = getConfig()

  client = new CriticClient(serverUrl)

  if (commentProvider) {
    commentProvider.dispose()
  }
  commentProvider = new CriticCommentProvider(client, () => showResolved)

  startPolling(pollIntervalMs)
}

function restartPolling(): void {
  stopPolling()
  commentProvider?.clearAll()
  initClient()
}

function startPolling(intervalMs: number): void {
  // Initial poll immediately
  void poll(true)
  pollTimer = setInterval(() => void poll(false), intervalMs)
}

function stopPolling(): void {
  if (pollTimer !== undefined) {
    clearInterval(pollTimer)
    pollTimer = undefined
  }
}

// ---- Polling --------------------------------------------------------------- //

async function poll(force: boolean): Promise<void> {
  if (!client || !commentProvider) return

  try {
    // Fetch git root once if not set
    if (!commentProvider['gitRoot']) {
      const cfg = await client.getConfig()
      commentProvider.setGitRoot(cfg.gitRoot)
      fileListProvider?.setGitRoot(cfg.gitRoot)
      baseFileProvider?.setGitRoot(cfg.gitRoot)
    }

    // Fetch diff bases for diff view support
    if (!currentStart) {
      try {
        const diffBases = await client.getDiffBases()
        currentStart = diffBases.currentStart
        fileListProvider?.setBaseRef(currentStart)
      } catch {
        // Server may not support GetDiffBases yet — continue without diff view
      }
    }

    // Check last change to avoid full fetch when nothing changed
    const changeRes = await client.getLastChange()
    const mtime = changeRes.mtimeMsecs

    if (!force && mtime === lastMtime) {
      // Nothing changed
      setStatus('connected')
      return
    }
    lastMtime = mtime

    const conversations = await client.getConversations()
    commentProvider.applyConversations(conversations)

    // Refresh file list in the sidebar
    if (fileListProvider) {
      void fileListProvider.refresh(client)
    }

    setStatus('connected', conversations)
    connected = true
  } catch (err) {
    if (connected) {
      // Only log on first disconnect to avoid spam
      console.error('Critic: connection error:', err)
    }
    connected = false
    setStatus('disconnected')
    commentProvider?.clearAll()
    fileListProvider?.clear()
  }
}

// ---- Status bar ------------------------------------------------------------ //

type ConnectionState = 'connected' | 'disconnected'

function setStatus(
  state: ConnectionState,
  conversations?: import('./criticClient').CriticConversation[]
): void {
  if (!statusBarItem) return

  if (state === 'disconnected') {
    statusBarItem.text = '$(comment-discussion) Critic $(debug-disconnect)'
    statusBarItem.tooltip = 'Critic: not connected — is the server running?\nClick for options'
    statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground')
    return
  }

  statusBarItem.backgroundColor = undefined

  if (!conversations || conversations.length === 0) {
    statusBarItem.text = '$(comment-discussion) Critic'
    statusBarItem.tooltip = 'Critic: connected — no open conversations\nClick for options'
    return
  }

  const unread = conversations.filter((c) =>
    c.messages.some((m) => m.isUnread)
  ).length

  const unresolved = conversations.filter(
    (c) =>
      c.status === 'CONVERSATION_STATUS_UNRESOLVED' ||
      c.status === 'CONVERSATION_STATUS_ACTIVE' ||
      c.status === 'CONVERSATION_STATUS_WAITING_FOR_RESPONSE'
  ).length

  const parts: string[] = []
  if (unread > 0) parts.push(`${unread} unread`)
  if (unresolved > 0) parts.push(`${unresolved} open`)

  statusBarItem.text =
    unread > 0
      ? `$(comment-discussion) Critic $(bell) ${parts.join(', ')}`
      : `$(comment-discussion) Critic ${parts.join(', ')}`

  statusBarItem.tooltip = [
    `Critic: ${unresolved} open conversation${unresolved !== 1 ? 's' : ''}`,
    unread > 0 ? `${unread} with unread AI replies` : '',
    'Click for options',
  ]
    .filter(Boolean)
    .join('\n')

  if (unread > 0) {
    statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.prominentBackground')
  } else {
    statusBarItem.backgroundColor = undefined
  }
}
