import { useEffect, useState, useCallback, useRef } from 'react'
import { criticClient, getConversationsSummary, getConversations, resolveConversation, ConversationSummary, CommentConversation } from '../api/client'
import { FileSummary, FileStatus, FileCategory } from '../gen/critic_pb'
import { pluralize } from './Pluralize'
import { matchesAnyPattern } from '../utils/glob'

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
  selectedFile: string | null
  onSelectFile: (file: string, fileSummary: FileSummary) => void
  onSelectRootConversation?: () => void
  isFocused?: boolean
  onFocus?: () => void
  filter: FilterType
  onFilterChange: (filter: FilterType) => void
  rootConversation?: CommentConversation | null
  isRootConversationSelected?: boolean
  onConversationsChanged?: () => void
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

// Convert a git pathspec glob pattern to a regex.
// Categorize a file based on project config categories.
// Categories are checked in order: the first matching category wins.
// Returns "source" if no category matches.
function categorizeFile(path: string, categories: FileCategory[]): string {
  for (const category of categories) {
    if (matchesAnyPattern(path, category.patterns)) {
      return category.name
    }
  }
  return 'source'
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

function FileList({ files, selectedFile, onSelectFile, onSelectRootConversation, isFocused, onFocus, filter, onFilterChange, rootConversation, isRootConversationSelected, onConversationsChanged }: FileListProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [categories, setCategories] = useState<FileCategory[]>([])
  const [conversationSummaries, setConversationSummaries] = useState<Map<string, ConversationSummary>>(new Map())
  const [fileConversations, setFileConversations] = useState<Map<string, CommentConversation[]>>(new Map())
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

  // Fetch project config (categories) once on mount
  useEffect(() => {
    criticClient.getProjectConfig({})
      .then((response) => {
        if (response.error) {
          console.error('Failed to fetch project config:', response.error.message)
        } else {
          setCategories(response.categories)
        }
        setLoading(false)
      })
      .catch((err) => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  // Fetch conversation summaries whenever files change (on reload)
  useEffect(() => {
    getConversationsSummary()
      .then((summaryResponse) => {
        const summaryMap = new Map<string, ConversationSummary>()
        for (const summary of summaryResponse.summaries) {
          summaryMap.set(summary.filePath, summary)
        }
        setConversationSummaries(summaryMap)
      })
      .catch((err) => {
        console.error('Failed to fetch conversation summaries:', err)
      })
  }, [files])

  // Fetch full conversations for files that have them (for conversations view)
  useEffect(() => {
    if (conversationSummaries.size === 0) return
    const filePaths = Array.from(conversationSummaries.entries())
      .filter(([, s]) => s.totalCount > 0)
      .map(([path]) => path)
    if (filePaths.length === 0) {
      setFileConversations(new Map())
      return
    }
    Promise.all(filePaths.map((path) => getConversations(path).then((r) => [path, r.conversations] as const)))
      .then((results) => {
        const map = new Map<string, CommentConversation[]>()
        for (const [path, convos] of results) {
          if (convos.length > 0) {
            map.set(path, convos)
          }
        }
        setFileConversations(map)
      })
  }, [conversationSummaries])

  // Scroll selected item into view
  useEffect(() => {
    if (selectedItemRef.current && isFocused) {
      selectedItemRef.current.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  }, [selectedFile, isFocused])

  // Compute visible, test, and hidden files based on project config categories, sorted by path
  const { regularFiles, testFiles, hiddenFiles } = (() => {
    const regular: FileSummary[] = []
    const tests: FileSummary[] = []
    const hidden: FileSummary[] = []
    for (const file of files) {
      const path = getFilePath(file)
      const category = categorizeFile(path, categories)
      if (category === 'hidden') {
        hidden.push(file)
      } else if (category === 'test') {
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

  // Total conversation count (from visible files only, plus root conversation if present)
  const hasRootConversation = rootConversation && rootConversation.messages.length > 0 ? 1 : 0
  const totalConversations = filesWithConversations.reduce((sum, file) => {
    const path = getFilePath(file)
    const summary = conversationSummaries.get(path)
    return sum + (summary?.totalCount || 0)
  }, hasRootConversation)

  // Compute effective filter: automatic means 'conversations' if any exist, otherwise 'files'
  const effectiveFilter = filter === 'automatic'
    ? (totalConversations > 0 ? 'conversations' : 'files')
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

  if (loading) {
    return (
      <div>
        <div className="file-list-header">Files (loading)</div>
        <div className="file-list-message">Loading...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div>
        <div className="file-list-header">Files (error)</div>
        <div className="file-list-message file-list-error">{error}</div>
      </div>
    )
  }

  return (
    <div className={`file-list-container${isFocused ? ' focused' : ''}`}>
      <div className="file-list-filters">
        <button
          className={`file-list-filter-btn${effectiveFilter === 'conversations' ? ' active' : ''}`}
          onClick={() => onFilterChange('conversations')}
        >
          {pluralize(totalConversations, 'Conversation')}
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
            {filesWithConversations.map((file) => {
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
                  {conversations.map((conv) => {
                    const lastMsg = conv.messages[conv.messages.length - 1]
                    const isUnresolved = conv.status !== 'resolved'
                    return (
                      <div
                        key={conv.id}
                        className={`conversation-entry${isUnresolved ? ' unresolved' : ''}`}
                        onClick={() => {
                          onSelectFile(path, file)
                          onFocus?.()
                        }}
                      >
                        <span className="conversation-entry-info">
                          <span className={`conversation-entry-status${isUnresolved ? ' unresolved' : ''}`}>
                            {isUnresolved ? 'open' : 'resolved'}
                          </span>
                          {lastMsg && <span className="conversation-entry-author">{lastMsg.author === 'human' ? 'Human' : 'Bot'}</span>}
                          <span className="conversation-entry-messages">
                            {pluralize(conv.messages.length, 'message')}
                          </span>
                          {lastMsg && <span className="conversation-entry-timestamp">{formatTimestamp(lastMsg.createdAt)}</span>}
                          {isUnresolved && (
                            <button
                              className="conversation-resolve-btn"
                              disabled={resolvingIds.has(conv.id)}
                              onClick={(e) => handleResolve(e, conv.id)}
                            >
                              {resolvingIds.has(conv.id) ? 'Resolving...' : 'Resolve'}
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
