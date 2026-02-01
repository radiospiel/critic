import { useState, useMemo, Fragment, useEffect, useCallback, useRef } from 'react'
import hljs from 'highlight.js'
import { FileDiff, FileStatus, Line, LineType } from '../gen/critic_pb'
import InlineCommentEditor, { CommentLineInfo } from './CommentEditor'
import CommentDisplay from './CommentDisplay'
import { getConversations, CommentConversation } from '../api/client'

const ALT_JUMP_SIZE = 25

interface DiffViewProps {
  fileDiff: FileDiff
  onNavigatePrevFile?: () => void
  onNavigateNextFile?: () => void
  isFocused?: boolean
  onFocus?: () => void
  contextLines?: number
  onIncreaseContext?: () => void
  onDecreaseContext?: () => void
  onResetContext?: () => void
  onSelectionChange?: (lineNoNew: number, lineNoOld: number) => void
  restoreLineNo?: { lineNoNew: number; lineNoOld: number } | null
  showOnlyConversations?: boolean
}

function getFileExtension(path: string): string {
  const parts = path.split('.')
  return parts.length > 1 ? parts[parts.length - 1] : ''
}

function getLanguage(path: string): string | undefined {
  const ext = getFileExtension(path).toLowerCase()

  // Handle special filenames without extensions
  const filename = path.split('/').pop()?.toLowerCase() || ''
  const filenameMap: Record<string, string> = {
    'dockerfile': 'dockerfile',
    'makefile': 'makefile',
    'gnumakefile': 'makefile',
    'cmakelists.txt': 'cmake',
    'rakefile': 'ruby',
    'gemfile': 'ruby',
    'podfile': 'ruby',
    'vagrantfile': 'ruby',
    'brewfile': 'ruby',
    '.gitignore': 'bash',
    '.dockerignore': 'bash',
    '.env': 'bash',
    '.env.local': 'bash',
    '.env.development': 'bash',
    '.env.production': 'bash',
  }

  if (filenameMap[filename]) {
    return filenameMap[filename]
  }

  const extMap: Record<string, string> = {
    // JavaScript/TypeScript
    ts: 'typescript',
    tsx: 'typescript',
    mts: 'typescript',
    cts: 'typescript',
    js: 'javascript',
    jsx: 'javascript',
    mjs: 'javascript',
    cjs: 'javascript',

    // Python
    py: 'python',
    pyw: 'python',
    pyi: 'python',

    // Go
    go: 'go',
    mod: 'go',

    // Rust
    rs: 'rust',

    // Ruby
    rb: 'ruby',
    erb: 'erb',
    rake: 'ruby',
    gemspec: 'ruby',

    // Java/JVM
    java: 'java',
    kt: 'kotlin',
    kts: 'kotlin',
    scala: 'scala',
    groovy: 'groovy',
    gradle: 'groovy',
    clj: 'clojure',
    cljs: 'clojure',
    cljc: 'clojure',

    // C/C++/Objective-C
    c: 'c',
    h: 'c',
    cpp: 'cpp',
    cxx: 'cpp',
    cc: 'cpp',
    hpp: 'cpp',
    hxx: 'cpp',
    hh: 'cpp',
    m: 'objectivec',
    mm: 'objectivec',

    // C#/F#
    cs: 'csharp',
    fs: 'fsharp',
    fsx: 'fsharp',
    fsi: 'fsharp',

    // Web
    html: 'xml',
    htm: 'xml',
    xhtml: 'xml',
    xml: 'xml',
    svg: 'xml',
    vue: 'xml',
    svelte: 'xml',
    astro: 'xml',

    // CSS/Styling
    css: 'css',
    scss: 'scss',
    sass: 'scss',
    less: 'less',
    styl: 'stylus',

    // Shell
    sh: 'bash',
    bash: 'bash',
    zsh: 'bash',
    fish: 'bash',
    ps1: 'powershell',
    psm1: 'powershell',
    bat: 'dos',
    cmd: 'dos',

    // Data formats
    json: 'json',
    jsonc: 'json',
    json5: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    toml: 'ini',
    ini: 'ini',
    cfg: 'ini',
    conf: 'ini',
    properties: 'properties',

    // Documentation
    md: 'markdown',
    markdown: 'markdown',
    rst: 'plaintext',
    txt: 'plaintext',
    tex: 'latex',

    // Database
    sql: 'sql',
    pgsql: 'pgsql',
    plsql: 'sql',

    // Other languages
    php: 'php',
    swift: 'swift',
    pl: 'perl',
    pm: 'perl',
    lua: 'lua',
    r: 'r',
    R: 'r',
    jl: 'julia',
    ex: 'elixir',
    exs: 'elixir',
    erl: 'erlang',
    hrl: 'erlang',
    hs: 'haskell',
    lhs: 'haskell',
    ml: 'ocaml',
    mli: 'ocaml',
    elm: 'elm',
    dart: 'dart',
    zig: 'zig',
    nim: 'nim',
    v: 'verilog',
    sv: 'verilog',
    vhd: 'vhdl',
    vhdl: 'vhdl',
    d: 'd',

    // Lisp family
    lisp: 'lisp',
    el: 'lisp',
    scm: 'scheme',
    rkt: 'scheme',

    // Config/DevOps
    proto: 'protobuf',
    graphql: 'graphql',
    gql: 'graphql',
    tf: 'hcl',
    hcl: 'hcl',
    nix: 'nix',
    dhall: 'haskell',

    // Assembly
    asm: 'x86asm',
    s: 'x86asm',

    // Misc
    coffee: 'coffeescript',
    diff: 'diff',
    patch: 'diff',
    sol: 'solidity',
    wasm: 'wasm',
    wat: 'wasm',
  }
  return extMap[ext]
}

function highlightLine(content: string, language: string | undefined): string {
  if (!language) {
    return escapeHtml(content)
  }
  try {
    const result = hljs.highlight(content, { language, ignoreIllegals: true })
    return result.value
  } catch {
    return escapeHtml(content)
  }
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

function parseHunkHeader(header: string): { context: string } | null {
  // Header format: @@ -10,5 +10,7 @@ optional context like function name
  const match = header.match(/^@@[^@]+@@\s*(.*)$/)
  if (match && match[1].trim()) {
    return { context: match[1].trim() }
  }
  return null
}

interface HunkHeaderProps {
  header: string
}

function HunkHeader({ header }: HunkHeaderProps) {
  const parsed = parseHunkHeader(header)
  if (parsed) {
    return (
      <>
        <span className="diff-hunk-label">Below </span>
        <code className="diff-hunk-context">{parsed.context}</code>
      </>
    )
  }
  return <code className="diff-hunk-context">{header}</code>
}

function getStatusDescription(status: FileStatus): string {
  switch (status) {
    case FileStatus.NEW:
      return 'Added'
    case FileStatus.DELETED:
      return 'Deleted'
    case FileStatus.RENAMED:
      return 'Renamed'
    case FileStatus.MODIFIED:
      return 'Modified'
    default:
      return 'Unknown'
  }
}

interface UnifiedLineProps {
  line: Line
  language: string | undefined
  isSelected?: boolean
  isFirstSelected?: boolean
  isLastSelected?: boolean
  lineRef?: React.RefObject<HTMLTableRowElement>
  onClick?: () => void
}

function UnifiedLine({ line, language, isSelected, isFirstSelected, isLastSelected, lineRef, onClick }: UnifiedLineProps) {
  const lineClass =
    line.type === LineType.ADDED
      ? 'diff-line-added'
      : line.type === LineType.DELETED
        ? 'diff-line-deleted'
        : 'diff-line-context'

  const prefix =
    line.type === LineType.ADDED ? '+' : line.type === LineType.DELETED ? '-' : ' '

  const highlightedContent = useMemo(
    () => highlightLine(line.content, language),
    [line.content, language]
  )

  const selectionClasses = [
    isSelected ? 'diff-line-selected' : '',
    isFirstSelected ? 'diff-line-selected-first' : '',
    isLastSelected ? 'diff-line-selected-last' : '',
  ].filter(Boolean).join(' ')

  return (
    <tr className={`${lineClass}${selectionClasses ? ' ' + selectionClasses : ''}`} ref={lineRef} onClick={onClick}>
      <td className="diff-line-number diff-line-number-old">
        {line.type !== LineType.ADDED && line.lineNoOld > 0 ? line.lineNoOld : ''}
      </td>
      <td className="diff-line-number diff-line-number-new">
        {line.type !== LineType.DELETED && line.lineNoNew > 0 ? line.lineNoNew : ''}
      </td>
      <td className="diff-line-prefix">{prefix}</td>
      <td
        className="diff-line-content"
        dangerouslySetInnerHTML={{ __html: highlightedContent || '&nbsp;' }}
      />
    </tr>
  )
}

// Selection range type
interface SelectionRange {
  start: number
  end: number
}

function DiffView({ fileDiff, onNavigatePrevFile, onNavigateNextFile, isFocused = true, onFocus, contextLines = 3, onIncreaseContext, onDecreaseContext, onResetContext, onSelectionChange, restoreLineNo, showOnlyConversations = false }: DiffViewProps) {
  const [selection, setSelection] = useState<SelectionRange>({ start: 0, end: 0 })
  const [editorOpen, setEditorOpen] = useState(false)
  const [comments, setComments] = useState<CommentConversation[]>([])
  const selectedLineRef = useRef<HTMLTableRowElement>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)

  const path = fileDiff.status === FileStatus.DELETED ? fileDiff.oldPath : fileDiff.newPath
  const language = getLanguage(path)
  const oldFile = fileDiff.oldPath
  const newFile = fileDiff.newPath

  // Fetch conversations when file changes
  const fetchComments = useCallback(async () => {
    if (!path) return
    try {
      const response = await getConversations(path)
      if (response.conversations) {
        setComments(response.conversations)
      }
    } catch (err) {
      console.error('Failed to fetch conversations:', err)
    }
  }, [path])

  useEffect(() => {
    fetchComments()
  }, [fetchComments])

  // Filter hunks to only show those with conversations when in conversations mode
  const filteredHunks = useMemo(() => {
    if (!showOnlyConversations || comments.length === 0) {
      return fileDiff.hunks
    }

    // Build a set of line numbers that have conversations
    const conversationLineNumbers = new Set(comments.map(c => c.lineNumber))

    // Filter hunks to only include those with at least one line that has a conversation
    return fileDiff.hunks.filter(hunk => {
      return hunk.lines.some(line => {
        // Check both old and new line numbers
        return conversationLineNumbers.has(line.lineNoNew) || conversationLineNumbers.has(line.lineNoOld)
      })
    })
  }, [fileDiff.hunks, showOnlyConversations, comments])

  // Count total navigable lines (all diff lines, excluding hunk headers)
  const totalLines = useMemo(() => {
    let count = 0
    for (const hunk of filteredHunks) {
      count += hunk.lines.length
    }
    return count
  }, [filteredHunks])

  // Build a flat array of all lines for easy indexing
  const allLines = useMemo(() => {
    const lines: { line: Line; hunkIdx: number; lineIdx: number }[] = []
    filteredHunks.forEach((hunk, hunkIdx) => {
      hunk.lines.forEach((line, lineIdx) => {
        lines.push({ line, hunkIdx, lineIdx })
      })
    })
    return lines
  }, [filteredHunks])

  // Get line info for the current selection (use the last selected line for positioning)
  const getSelectionLineInfo = useCallback((): CommentLineInfo | null => {
    if (allLines.length === 0) return null
    const lastSelectedIdx = Math.min(selection.end, allLines.length - 1)
    const { line } = allLines[lastSelectedIdx]
    return {
      oldFile,
      newFile,
      oldLine: line.lineNoOld,
      newLine: line.lineNoNew,
    }
  }, [allLines, selection.end, oldFile, newFile])

  const stats = useMemo(() => {
    let added = 0
    let deleted = 0
    for (const hunk of fileDiff.hunks) {
      added += hunk.stats?.added || 0
      deleted += hunk.stats?.deleted || 0
    }
    return { added, deleted }
  }, [fileDiff.hunks])

  // Reset selection when file changes, trying to restore to the target line if provided
  useEffect(() => {
    setEditorOpen(false)

    if (restoreLineNo && allLines.length > 0) {
      // Try to find the exact line by lineNoNew first, then lineNoOld
      let bestIndex = -1
      let bestDistance = Infinity

      for (let i = 0; i < allLines.length; i++) {
        const { line } = allLines[i]
        // Exact match on new line number
        if (line.lineNoNew === restoreLineNo.lineNoNew && restoreLineNo.lineNoNew > 0) {
          bestIndex = i
          break
        }
        // Exact match on old line number
        if (line.lineNoOld === restoreLineNo.lineNoOld && restoreLineNo.lineNoOld > 0) {
          bestIndex = i
          break
        }
        // Track closest line by new line number
        if (line.lineNoNew > 0 && restoreLineNo.lineNoNew > 0) {
          const distance = Math.abs(line.lineNoNew - restoreLineNo.lineNoNew)
          if (distance < bestDistance) {
            bestDistance = distance
            bestIndex = i
          }
        }
        // Track closest line by old line number
        if (line.lineNoOld > 0 && restoreLineNo.lineNoOld > 0) {
          const distance = Math.abs(line.lineNoOld - restoreLineNo.lineNoOld)
          if (distance < bestDistance) {
            bestDistance = distance
            bestIndex = i
          }
        }
      }

      if (bestIndex >= 0) {
        setSelection({ start: bestIndex, end: bestIndex })
        return
      }
    }

    setSelection({ start: 0, end: 0 })
  }, [fileDiff, restoreLineNo, allLines])

  // Scroll selected line into view and notify parent of selection change
  useEffect(() => {
    if (selectedLineRef.current) {
      selectedLineRef.current.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
    // Notify parent of the current selection's line numbers
    if (onSelectionChange && allLines.length > 0 && selection.end < allLines.length) {
      const { line } = allLines[selection.end]
      onSelectionChange(line.lineNoNew, line.lineNoOld)
    }
  }, [selection, allLines, onSelectionChange])

  // Check if a line index is within the selection range
  const isLineSelected = useCallback(
    (lineIndex: number) => lineIndex >= selection.start && lineIndex <= selection.end,
    [selection]
  )

  const handleEditorClose = useCallback(() => {
    setEditorOpen(false)
  }, [])

  const handleCommentSaved = useCallback(() => {
    console.log('Comment saved successfully')
    setEditorOpen(false)
    // Refresh comments after saving
    fetchComments()
  }, [fetchComments])

  // Get comments for a specific line number
  const getCommentsForLine = useCallback(
    (lineNo: number) => comments.filter((c) => c.lineNumber === lineNo),
    [comments]
  )

  // Keyboard navigation
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Don't handle if not focused or in an input field
      if (!isFocused) return
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return
      }
      // Don't handle if in tiptap editor
      if ((e.target as HTMLElement)?.closest?.('.tiptap')) {
        return
      }

      const jumpSize = e.altKey ? ALT_JUMP_SIZE : 1

      switch (e.key) {
        case 'Enter':
          if (!editorOpen) {
            e.preventDefault()
            setEditorOpen(true)
          }
          break
        case 'Escape':
          if (editorOpen) {
            e.preventDefault()
            setEditorOpen(false)
          }
          break
        case 'ArrowUp':
        case 'k':
          e.preventDefault()
          if (editorOpen) return // Don't navigate while editor is open
          if (e.shiftKey) {
            // Expand selection upward
            if (selection.start > 0) {
              setSelection((prev) => ({
                ...prev,
                start: Math.max(0, prev.start - 1),
              }))
            }
          } else {
            // Collapse to top of selection and move up
            const newIndex = Math.max(0, selection.start - jumpSize)
            if (selection.start === 0 && onNavigatePrevFile) {
              onNavigatePrevFile()
            } else {
              setSelection({ start: newIndex, end: newIndex })
            }
          }
          break
        case 'ArrowDown':
        case 'j':
          e.preventDefault()
          if (editorOpen) return // Don't navigate while editor is open
          if (e.shiftKey) {
            // Expand selection downward
            if (selection.end < totalLines - 1) {
              setSelection((prev) => ({
                ...prev,
                end: Math.min(totalLines - 1, prev.end + 1),
              }))
            }
          } else {
            // Collapse to bottom of selection and move down
            const newIndex = Math.min(totalLines - 1, selection.end + jumpSize)
            if (selection.end >= totalLines - 1 && onNavigateNextFile) {
              onNavigateNextFile()
            } else {
              setSelection({ start: newIndex, end: newIndex })
            }
          }
          break
        case 'g':
          if (!e.shiftKey && !editorOpen) {
            e.preventDefault()
            setSelection({ start: 0, end: 0 })
          }
          break
        case 'G':
          if (!editorOpen) {
            e.preventDefault()
            setSelection({ start: totalLines - 1, end: totalLines - 1 })
          }
          break
        case 'c':
          if (!editorOpen && onIncreaseContext) {
            e.preventDefault()
            onIncreaseContext()
          }
          break
        case 'C':
          if (!editorOpen && onDecreaseContext) {
            e.preventDefault()
            onDecreaseContext()
          }
          break
      }
    },
    [isFocused, selection, totalLines, onNavigatePrevFile, onNavigateNextFile, editorOpen, onIncreaseContext, onDecreaseContext]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  if (fileDiff.isBinary) {
    return (
      <div className={`diff-view${isFocused ? ' focused' : ''}`}>
        <div className="diff-header">
          <div className="diff-header-info">
            <span className="diff-file-path">{path}</span>
            <span className="diff-file-status">{getStatusDescription(fileDiff.status)}</span>
          </div>
        </div>
        <div className="diff-binary-notice">Binary file not shown</div>
      </div>
    )
  }

  const selectionLineInfo = getSelectionLineInfo()

  return (
    <div className={`diff-view${isFocused ? ' focused' : ''}`}>
      <div className="diff-header">
        <div className="diff-header-info">
          <span className="diff-file-path">{path}</span>
          <span className="diff-file-status">{getStatusDescription(fileDiff.status)}</span>
          {fileDiff.status === FileStatus.RENAMED && fileDiff.oldPath !== fileDiff.newPath && (
            <span className="diff-renamed-from">from {fileDiff.oldPath}</span>
          )}
        </div>
        <div className="diff-header-actions">
          <span className="diff-stats">
            <span className="diff-stats-added">+{stats.added}</span>
            <span className="diff-stats-deleted">-{stats.deleted}</span>
          </span>
          <div className="diff-context-controls">
            <button
              className="diff-context-button"
              onClick={onDecreaseContext}
              disabled={contextLines <= 3}
              title="Decrease context lines (Shift+C)"
            >
              −C
            </button>
            <button
              className="diff-context-button"
              onClick={onResetContext}
              disabled={contextLines === 3}
              title="Reset to default context (3 lines)"
            >
              C
            </button>
            <button
              className="diff-context-button"
              onClick={onIncreaseContext}
              title="Increase context lines (c)"
            >
              +C
            </button>
          </div>
        </div>
      </div>

      <div className="diff-content" ref={containerRef}>
        {filteredHunks.length === 0 ? (
          <div className="diff-empty-notice">{showOnlyConversations ? 'No conversations in this file (you probably have outdated conversation data, this will be fixed automatically.)' : 'No changes in this file'}</div>
        ) : (
          <table className="diff-table diff-table-unified">
            <tbody>
              {(() => {
                let globalLineIndex = 0
                return filteredHunks.map((hunk, hunkIdx) => (
                  <Fragment key={hunkIdx}>
                    <tr className="diff-hunk-header-row">
                      <td colSpan={4} className="diff-hunk-header">
                        <HunkHeader header={hunk.header} />
                      </td>
                    </tr>
                    {hunk.lines.map((line, lineIdx) => {
                      const currentIndex = globalLineIndex++
                      const isSelected = isLineSelected(currentIndex)
                      const isFirstSelected = currentIndex === selection.start
                      const isLastSelected = currentIndex === selection.end
                      // Attach ref to the last selected line for scroll-into-view
                      const shouldAttachRef = isLastSelected
                      // Show editor after the last selected line
                      const showEditorAfterLine = isLastSelected && editorOpen && selectionLineInfo
                      // Get comments for this line
                      // For deleted lines: use lineNoOld, but skip if next line is ADDED (modified line - show on added instead)
                      // For added/context lines: use lineNoNew
                      const isDeletedFollowedByAdded = line.type === LineType.DELETED &&
                        lineIdx + 1 < hunk.lines.length &&
                        hunk.lines[lineIdx + 1].type === LineType.ADDED
                      const lineNo = line.type === LineType.DELETED ? line.lineNoOld : line.lineNoNew
                      const lineComments = isDeletedFollowedByAdded ? [] : getCommentsForLine(lineNo)
                      const hasComments = lineComments.length > 0

                      return (
                        <Fragment key={`${hunkIdx}-${lineIdx}`}>
                          <UnifiedLine
                            line={line}
                            language={language}
                            isSelected={isSelected}
                            isFirstSelected={isFirstSelected}
                            isLastSelected={isLastSelected}
                            lineRef={shouldAttachRef ? selectedLineRef : undefined}
                            onClick={() => {
                              setSelection({ start: currentIndex, end: currentIndex })
                              onFocus?.()
                            }}
                          />
                          {hasComments && (
                            <tr className="diff-comment-row">
                              <td colSpan={4}>
                                <div className="diff-comment-wrapper">
                                  <CommentDisplay conversations={lineComments} lineNumber={lineNo} onReplyAdded={fetchComments} />
                                </div>
                              </td>
                            </tr>
                          )}
                          {showEditorAfterLine && (
                            <tr className="diff-inline-editor-row">
                              <td colSpan={4}>
                                <div className="diff-comment-wrapper">
                                  <InlineCommentEditor
                                    lineInfo={selectionLineInfo}
                                    onClose={handleEditorClose}
                                    onSaved={handleCommentSaved}
                                  />
                                </div>
                              </td>
                            </tr>
                          )}
                        </Fragment>
                      )
                    })}
                  </Fragment>
                ))
              })()}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export default DiffView
