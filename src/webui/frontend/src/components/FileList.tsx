import { useEffect, useState, useCallback, useRef, useMemo } from 'react'
import { resolveConversation, archiveConversation, unresolveConversation, ConversationSummary, CommentConversation } from '../api/client'
import { FileSummary, FileStatus } from '../gen/critic_pb'
import { pluralize } from './Pluralize'

function formatTimestamp(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)
  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  return date.toLocaleDateString()
}

export type FilterType = 'automatic' | 'conversations' | 'files' | 'tests' | 'hidden'

interface FileListProps {
  files: FileSummary[]
  allConversations: CommentConversation[]
  selectedFile: string | null
  onSelectFile: (file: string, fileSummary: FileSummary) => void
  onSelectRootConversation?: () => void
  isFocused?: boolean
  onFocus?: () => void
  filter: FilterType
  onFilterChange: (filter: FilterType) => void
  showArchived?: boolean
  rootConversation?: CommentConversation | null
  isRootConversationSelected?: boolean
  onConversationsChanged?: () => void
  onScrollToLine?: (lineNo: number) => void
}

function getFilePath(file: FileSummary): string {
  return file.status === FileStatus.DELETED ? file.oldPath : file.newPath
}

function getStatusLabel(status: FileStatus): string {
  switch (status) {
    case FileStatus.NEW:
      return 'A'
    case FileStatus.DELETED:
      return 'D'
    case FileStatus.RENAMED:
      return 'R'
    case FileStatus.UNTRACKED:
      return '?'
    case FileStatus.MODIFIED:
    default:
      return 'M'
  }
}

// Truncate text to maxLen characters, adding ellipsis if needed
function truncateText(text: string, maxLen: number): string {
  // Strip markdown/HTML and collapse whitespace
  const plain = text.replace(/[#*_`~\[\]()>]/g, '').replace(/\s+/g, ' ').trim()
  if (plain.length <= maxLen) return plain
  return plain.slice(0, maxLen) + '...'
}

interface CommentMessage {
  id?: string
  author: string
  content: string
  createdAt: string
}

function MessagePreviews({ messages }: { messages: CommentMessage[] }) {
  if (messages.length === 0) return null
  const total = messages.length
  const tail = messages.slice(-2)
  return (
    <>
      {total > 2 && (
        <span className="conversation-entry-ellipsis">
          ({pluralize(total - 2, 'earlier message')})
        </span>
      )}
      {tail.map((msg, i) => (
        <span key={i} className="conversation-entry-preview">
          <span className="conversation-entry-preview-author">{msg.author === 'human' ? 'Human' : 'Bot'}:</span>{' '}
          {truncateText(msg.content, 150)}
        </span>
      ))}
    </>
  )
}

function FileList({ files, allConversations, selectedFile, onSelectFile, onSelectRootConversation, isFocused, onFocus, filter, onFilterChange, showArchived = false, rootConversation, isRootConversationSelected, onConversationsChanged, onScrollToLine }: FileListProps) {
  const [resolvingIds, setResolvingIds] = useState<Set<string>>(new Set())
  const selectedItemRef = useRef<HTMLLIElement>(null)

  const handleResolve = async (e: React.MouseEvent, conversationId: string) => {
    e.stopPropagation()
    setResolvingIds((prev) => new Set(prev).add(conversationId))
    const result = await resolveConversation(conversationId)
    setResolvingIds((prev) => {
      const next = new Set(prev)
      next.delete(conversationId)
      return next
    })
    if (result.success) {
      onConversationsChanged?.()
    }
  }

  const handleArchive = async (e: React.MouseEvent, conversationId: string) => {
    e.stopPropagation()
    setResolvingIds((prev) => new Set(prev).add(conversationId))
    const result = await archiveConversation(conversationId)
    setResolvingIds((prev) => {
      const next = new Set(prev)
      next.delete(conversationId)
      return next
    })
    if (result.success) {
      onConversationsChanged?.()
    }
  }

  const handleUnresolve = async (e: React.MouseEvent, conversationId: string) => {
    e.stopPropagation()
    setResolvingIds((prev) => new Set(prev).add(conversationId))
    const result = await unresolveConversation(conversationId)
    setResolvingIds((prev) => {
      const next = new Set(prev)
      next.delete(conversationId)
      return next
    })
    if (result.success) {
      onConversationsChanged?.()
    }
  }

  // Derive per-file conversation and summary maps from the single bulk load
  const { fileConversations, conversationSummaries } = useMemo(() => {
    const convMap = new Map<string, CommentConversation[]>()
    const summaryMap = new Map<string, ConversationSummary>()

    for (const conv of allConversations) {
      const path = conv.filePath
      if (!convMap.has(path)) {
        convMap.set(path, [])
      }
      convMap.get(path)!.push(conv)
    }

    for (const [path, convos] of convMap) {
      summaryMap.set(path, {
        filePath: path,
        totalCount: convos.length,
        unresolvedCount: convos.filter((c) => c.status !== 'resolved' && c.status !== 'informal').length,
        resolvedCount: convos.filter((c) => c.status === 'resolved').length,
        explanationCount: convos.filter((c) => c.conversationType === 'explanation').length,
        hasUnreadAiMessages: convos.some((c) => c.messages.some((m) => m.isUnread)),
      })
    }

    return { fileConversations: convMap, conversationSummaries: summaryMap }
  }, [allConversations])

  // Scroll selected item into view
  useEffect(() => {
    if (selectedItemRef.current && isFocused) {
      selectedItemRef.current.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  }, [selectedFile, isFocused])

  // Compute visible, test, and hidden files using server-provided category, sorted by path
  const { regularFiles, testFiles, hiddenFiles } = (() => {
    const regular: FileSummary[] = []
    const tests: FileSummary[] = []
    const hidden: FileSummary[] = []
    for (const file of files) {
      if (file.category === 'hidden') {
        hidden.push(file)
      } else if (file.category === 'test') {
        tests.push(file)
      } else {
        regular.push(file)
      }
    }
    // Sort each category by path
    const sortByPath = (a: FileSummary, b: FileSummary) =>
      getFilePath(a).localeCompare(getFilePath(b))
    return {
      regularFiles: regular.sort(sortByPath),
      testFiles: tests.sort(sortByPath),
      hiddenFiles: hidden.sort(sortByPath),
    }
  })()

  // All visible files (regular + tests) for conversations filter
  const visibleFiles = [...regularFiles, ...testFiles]

  // Files with conversations (from visible files only), sorted by path
  const filesWithConversations = visibleFiles
    .filter((file) => {
      const path = getFilePath(file)
      const summary = conversationSummaries.get(path)
      return summary && summary.totalCount > 0
    })
    .sort((a, b) => getFilePath(a).localeCompare(getFilePath(b)))

  // Total conversation count: only non-resolved, non-archived items (matching what's displayed)
  const hasRootConversation = rootConversation && rootConversation.messages.length > 0 && rootConversation.status !== 'resolved' && rootConversation.status !== 'archived' ? 1 : 0
  const totalVisibleConversations = hasRootConversation + Array.from(fileConversations.values()).reduce((sum, convos) => {
    return sum + convos.filter((c) => c.status !== 'resolved' && (showArchived || c.status !== 'archived')).length
  }, 0)

  // Compute effective filter: automatic means 'conversations' if any exist, otherwise 'files'
  const effectiveFilter = filter === 'automatic'
    ? (totalVisibleConversations > 0 ? 'conversations' : 'files')
    : filter

  // Determine displayed files based on filter
  const displayedFiles = (() => {
    switch (effectiveFilter) {
      case 'conversations':
        return filesWithConversations
      case 'tests':
        return testFiles
      case 'hidden':
        return hiddenFiles
      case 'files':
      default:
        return regularFiles
    }
  })()

  // Keyboard navigation when focused
  const showRootInList = effectiveFilter === 'conversations' && hasRootConversation > 0

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isFocused) return

      // Don't handle if in input field
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return
      }

      const currentIndex = selectedFile
        ? displayedFiles.findIndex((f) => getFilePath(f) === selectedFile)
        : -1
      const onRoot = isRootConversationSelected

      switch (e.key) {
        case 'ArrowUp':
        case 'k':
          e.preventDefault()
          if (onRoot) {
            // Already at the top, do nothing
          } else if (currentIndex === 0 && showRootInList) {
            // At first file, go up to root conversation
            onSelectRootConversation?.()
          } else if (currentIndex > 0) {
            const prevFile = displayedFiles[currentIndex - 1]
            onSelectFile(getFilePath(prevFile), prevFile)
          } else if (currentIndex === -1 && !onRoot) {
            // No selection, select last file
            if (displayedFiles.length > 0) {
              const lastFile = displayedFiles[displayedFiles.length - 1]
              onSelectFile(getFilePath(lastFile), lastFile)
            } else if (showRootInList) {
              onSelectRootConversation?.()
            }
          }
          break
        case 'ArrowDown':
        case 'j':
          e.preventDefault()
          if (onRoot && displayedFiles.length > 0) {
            // On root, go down to first file
            const firstFile = displayedFiles[0]
            onSelectFile(getFilePath(firstFile), firstFile)
          } else if (currentIndex < displayedFiles.length - 1) {
            const nextFile = displayedFiles[currentIndex + 1]
            onSelectFile(getFilePath(nextFile), nextFile)
          } else if (currentIndex === -1 && !onRoot) {
            // No selection, select root or first file
            if (showRootInList) {
              onSelectRootConversation?.()
            } else if (displayedFiles.length > 0) {
              const firstFile = displayedFiles[0]
              onSelectFile(getFilePath(firstFile), firstFile)
            }
          }
          break
        case 'Enter':
          // Already selected, just confirm (could switch focus to diff view)
          break
      }
    },
    [isFocused, displayedFiles, selectedFile, onSelectFile, isRootConversationSelected, showRootInList, onSelectRootConversation]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  return (
    <div className={`file-list-container${isFocused ? ' focused' : ''}`}>
      <div className="file-list-filters">
        <button
          className={`file-list-filter-btn${effectiveFilter === 'conversations' ? ' active' : ''}`}
          onClick={() => onFilterChange('conversations')}
        >
          {pluralize(totalVisibleConversations, 'Conversation')}
        </button>
        <button
          className={`file-list-filter-btn${effectiveFilter === 'files' ? ' active' : ''}`}
          onClick={() => onFilterChange('files')}
        >
          {pluralize(regularFiles.length, 'File')}
        </button>
        <button
          className={`file-list-filter-btn${effectiveFilter === 'tests' ? ' active' : ''}`}
          onClick={() => onFilterChange('tests')}
        >
          {pluralize(testFiles.length, 'Test')}
        </button>
        <button
          className={`file-list-filter-btn${filter === 'hidden' ? ' active' : ''}`}
          onClick={() => onFilterChange('hidden')}
        >
          {hiddenFiles.length} Hidden
        </button>
      </div>
      <ul className="file-list">
        {effectiveFilter === 'conversations' ? (
          <>
            {rootConversation && rootConversation.messages.length > 0 && (() => {
              const lastMsg = rootConversation.messages[rootConversation.messages.length - 1]
              const isUnresolved = rootConversation.status !== 'resolved'
              return (
                <li
                  className={`conversation-entry root${isUnresolved ? ' unresolved' : ''}${isRootConversationSelected ? ' selected' : ''}`}
                  onClick={() => {
                    onSelectRootConversation?.()
                    onFocus?.()
                  }}
                >
                  <span className="conversation-entry-info">
                    <span className={`conversation-entry-status${isUnresolved ? ' unresolved' : ''}`}>
                      {isUnresolved ? 'open' : 'resolved'}
                    </span>
                    <span className="conversation-entry-author">{lastMsg.author === 'human' ? 'Human' : 'Bot'}</span>
                    <span className="conversation-entry-messages">
                      {pluralize(rootConversation.messages.length, 'message')}
                    </span>
                    <span className="conversation-entry-timestamp">{formatTimestamp(lastMsg.createdAt)}</span>
                    {isUnresolved && (
                      <button
                        className="conversation-resolve-btn"
                        disabled={resolvingIds.has(rootConversation.id)}
                        onClick={(e) => handleResolve(e, rootConversation.id)}
                      >
                        {resolvingIds.has(rootConversation.id) ? 'Resolving...' : 'Resolve'}
                      </button>
                    )}
                  </span>
                  <MessagePreviews messages={rootConversation.messages} />
                </li>
              )
            })()}
            {filesWithConversations.filter((file) => {
              const convos = fileConversations.get(getFilePath(file)) || []
              return convos.some((c) => c.status !== 'resolved' && (showArchived || c.status !== 'archived'))
            }).map((file) => {
              const path = getFilePath(file)
              const conversations = fileConversations.get(path) || []
              const isSelected = selectedFile === path
              const summary = conversationSummaries.get(path)
              const hasUnresolved = summary ? summary.unresolvedCount > 0 : false
              return (
                <li
                  key={path}
                  ref={isSelected ? selectedItemRef : undefined}
                  className="conversation-group"
                >
                  <div
                    className={`conversation-group-header${isSelected ? ' selected' : ''}${hasUnresolved ? ' unresolved' : ''}`}
                    onClick={() => {
                      onSelectFile(path, file)
                      onFocus?.()
                    }}
                    title={path}
                  >
                    <span className={`file-status status-${getStatusLabel(file.status).toLowerCase()}`}>
                      {getStatusLabel(file.status)}
                    </span>
                    <span className="file-path">{path}</span>
                  </div>
                  {conversations.filter((c) => !(c.conversationType === 'explanation' && c.status === 'resolved')).filter((c) => showArchived || c.status !== 'archived').map((conv) => {
                    const lastMsg = conv.messages[conv.messages.length - 1]
                    const isUnresolved = conv.status !== 'resolved' && conv.status !== 'informal'
                    const isExplanation = conv.conversationType === 'explanation'
                    return (
                      <div
                        key={conv.id}
                        className={`conversation-entry${isUnresolved ? ' unresolved' : ''}${isExplanation ? ' explanation' : ''}`}
                        onClick={() => {
                          onSelectFile(path, file)
                          onScrollToLine?.(conv.lineNumber)
                          onFocus?.()
                        }}
                      >
                        <span className="conversation-entry-info">
                          {isExplanation && (
                            <svg className="explanation-icon" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                              <path d="M8 1.5c-2.363 0-4 1.69-4 3.75 0 .984.424 1.625.984 2.304l.214.253c.223.264.47.556.673.848.284.411.537.896.621 1.49a.75.75 0 0 1-1.484.211c-.04-.282-.163-.547-.37-.847a8.456 8.456 0 0 0-.542-.68c-.084-.1-.173-.205-.268-.32C3.201 7.75 2.5 6.766 2.5 5.25 2.5 2.31 4.863.5 8 .5s5.5 1.81 5.5 4.75c0 1.516-.701 2.5-1.328 3.259-.095.115-.184.22-.268.319-.207.245-.383.453-.541.681-.208.3-.33.565-.37.847a.751.751 0 0 1-1.485-.212c.084-.593.337-1.078.621-1.489.203-.292.45-.584.673-.848.075-.088.147-.173.213-.253.561-.679.985-1.32.985-2.304 0-2.06-1.637-3.75-4-3.75ZM5.75 12h4.5a.75.75 0 0 1 0 1.5h-4.5a.75.75 0 0 1 0-1.5ZM6 15.25a.75.75 0 0 1 .75-.75h2.5a.75.75 0 0 1 0 1.5h-2.5a.75.75 0 0 1-.75-.75Z"/>
                            </svg>
                          )}
                          <span className={`conversation-entry-status${isUnresolved ? ' unresolved' : ''}`}>
                            {isExplanation ? 'explain' : isUnresolved ? 'open' : 'resolved'}
                          </span>
                          {lastMsg && <span className="conversation-entry-author">{lastMsg.author === 'human' ? 'Human' : 'Bot'}</span>}
                          <span className="conversation-entry-messages">
                            {pluralize(conv.messages.length, 'message')}
                          </span>
                          {lastMsg && <span className="conversation-entry-timestamp">{formatTimestamp(lastMsg.createdAt)}</span>}
                          {(isUnresolved || (isExplanation && conv.status !== 'resolved')) && (
                            <button
                              className="conversation-resolve-btn"
                              disabled={resolvingIds.has(conv.id)}
                              onClick={(e) => handleResolve(e, conv.id)}
                            >
                              {resolvingIds.has(conv.id) ? '...' : 'Resolve'}
                            </button>
                          )}
                          {conv.status === 'resolved' && (
                            <button
                              className="conversation-resolve-btn"
                              disabled={resolvingIds.has(conv.id)}
                              onClick={(e) => handleUnresolve(e, conv.id)}
                            >
                              {resolvingIds.has(conv.id) ? '...' : 'Unresolve'}
                            </button>
                          )}
                          {conv.status !== 'archived' && (
                            <button
                              className="conversation-resolve-btn"
                              disabled={resolvingIds.has(conv.id)}
                              onClick={(e) => handleArchive(e, conv.id)}
                            >
                              {resolvingIds.has(conv.id) ? '...' : 'Archive'}
                            </button>
                          )}
                        </span>
                        <MessagePreviews messages={conv.messages} />
                      </div>
                    )
                  })}
                </li>
              )
            })}
            {filesWithConversations.length === 0 && (!rootConversation || rootConversation.messages.length === 0) && (
              <li className="file-list-message">No conversations yet</li>
            )}
          </>
        ) : (
          <>
            {displayedFiles.map((file) => {
              const path = getFilePath(file)
              const isSelected = selectedFile === path
              const summary = conversationSummaries.get(path)
              return (
                <li
                  key={path}
                  ref={isSelected ? selectedItemRef : undefined}
                  className={`file-item${isSelected ? ' selected' : ''}`}
                  onClick={() => {
                    onSelectFile(path, file)
                    onFocus?.()
                  }}
                  title={path}
                >
                  <span className={`file-status status-${getStatusLabel(file.status).toLowerCase()}`}>
                    {getStatusLabel(file.status)}
                  </span>
                  <span className="file-path">{path}</span>
                  {summary && summary.unresolvedCount > 0 && (
                    <span
                      className="conversation-icon unresolved"
                      title={`${summary.unresolvedCount} open`}
                    >
                      {summary.unresolvedCount}
                    </span>
                  )}
                  {summary && summary.resolvedCount > 0 && (
                    <span
                      className="conversation-icon resolved"
                      title={`${summary.resolvedCount} resolved`}
                    >
                      {summary.resolvedCount}
                    </span>
                  )}
                </li>
              )
            })}
            {displayedFiles.length === 0 && (
              <li className="file-list-message">
                {filter === 'conversations'
                  ? 'No files with conversations'
                  : filter === 'tests'
                    ? 'No test files (configure via project.critic)'
                    : filter === 'hidden'
                      ? 'Files can be hidden via project.critic'
                      : 'No files found'}
              </li>
            )}
          </>
        )}
      </ul>
    </div>
  )
}

export default FileList
