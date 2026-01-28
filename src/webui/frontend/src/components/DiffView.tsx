import { useState, useMemo, Fragment } from 'react'
import hljs from 'highlight.js'
import { FileDiff, FileStatus, Hunk, Line, LineType } from '../gen/critic_pb'

type ViewMode = 'unified' | 'split'

interface DiffViewProps {
  fileDiff: FileDiff
}

function getFileExtension(path: string): string {
  const parts = path.split('.')
  return parts.length > 1 ? parts[parts.length - 1] : ''
}

function getLanguage(path: string): string | undefined {
  const ext = getFileExtension(path)
  const extMap: Record<string, string> = {
    ts: 'typescript',
    tsx: 'typescript',
    js: 'javascript',
    jsx: 'javascript',
    py: 'python',
    go: 'go',
    rs: 'rust',
    rb: 'ruby',
    java: 'java',
    c: 'c',
    cpp: 'cpp',
    h: 'c',
    hpp: 'cpp',
    cs: 'csharp',
    css: 'css',
    scss: 'scss',
    html: 'html',
    xml: 'xml',
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    md: 'markdown',
    sh: 'bash',
    bash: 'bash',
    sql: 'sql',
    proto: 'protobuf',
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
}

function UnifiedLine({ line, language }: UnifiedLineProps) {
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

  return (
    <tr className={lineClass}>
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

interface SplitViewProps {
  hunks: Hunk[]
  language: string | undefined
}

interface SplitLine {
  oldLine: Line | null
  newLine: Line | null
}

function computeSplitLines(hunks: Hunk[]): SplitLine[] {
  const result: SplitLine[] = []

  for (const hunk of hunks) {
    // Add hunk header as a separator
    result.push({
      oldLine: null,
      newLine: null,
    })

    let i = 0
    while (i < hunk.lines.length) {
      const line = hunk.lines[i]

      if (line.type === LineType.CONTEXT) {
        result.push({ oldLine: line, newLine: line })
        i++
      } else if (line.type === LineType.DELETED) {
        // Collect consecutive deleted lines
        const deletedLines: Line[] = []
        while (i < hunk.lines.length && hunk.lines[i].type === LineType.DELETED) {
          deletedLines.push(hunk.lines[i])
          i++
        }
        // Collect consecutive added lines
        const addedLines: Line[] = []
        while (i < hunk.lines.length && hunk.lines[i].type === LineType.ADDED) {
          addedLines.push(hunk.lines[i])
          i++
        }
        // Pair up deleted and added lines
        const maxLen = Math.max(deletedLines.length, addedLines.length)
        for (let j = 0; j < maxLen; j++) {
          result.push({
            oldLine: j < deletedLines.length ? deletedLines[j] : null,
            newLine: j < addedLines.length ? addedLines[j] : null,
          })
        }
      } else if (line.type === LineType.ADDED) {
        // Added line without preceding deleted line
        result.push({ oldLine: null, newLine: line })
        i++
      } else {
        i++
      }
    }
  }

  return result
}

function SplitView({ hunks, language }: SplitViewProps) {
  const splitLines = useMemo(() => computeSplitLines(hunks), [hunks])

  return (
    <table className="diff-table diff-table-split">
      <tbody>
        {splitLines.map((pair, idx) => {
          // Hunk separator
          if (pair.oldLine === null && pair.newLine === null) {
            const hunkIndex = hunks.findIndex((_, hi) => {
              let count = 0
              for (let j = 0; j <= hi; j++) {
                count++ // for hunk separator
                count += hunks[j].lines.length
              }
              return idx < count
            })
            const hunk = hunks[hunkIndex] || hunks[0]
            return (
              <tr key={idx} className="diff-hunk-header-row">
                <td colSpan={4} className="diff-hunk-header">
                  <HunkHeader header={hunk?.header || '@@'} />
                </td>
              </tr>
            )
          }

          const oldClass = pair.oldLine
            ? pair.oldLine.type === LineType.DELETED
              ? 'diff-line-deleted'
              : 'diff-line-context'
            : 'diff-line-empty'
          const newClass = pair.newLine
            ? pair.newLine.type === LineType.ADDED
              ? 'diff-line-added'
              : 'diff-line-context'
            : 'diff-line-empty'

          const oldContent = pair.oldLine
            ? highlightLine(pair.oldLine.content, language)
            : ''
          const newContent = pair.newLine
            ? highlightLine(pair.newLine.content, language)
            : ''

          return (
            <tr key={idx} className="diff-split-row">
              <td className={`diff-line-number ${oldClass}`}>
                {pair.oldLine?.lineNoOld || ''}
              </td>
              <td
                className={`diff-split-content ${oldClass}`}
                dangerouslySetInnerHTML={{ __html: oldContent || '&nbsp;' }}
              />
              <td className={`diff-line-number ${newClass}`}>
                {pair.newLine?.lineNoNew || ''}
              </td>
              <td
                className={`diff-split-content ${newClass}`}
                dangerouslySetInnerHTML={{ __html: newContent || '&nbsp;' }}
              />
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

function DiffView({ fileDiff }: DiffViewProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('unified')

  const path = fileDiff.status === FileStatus.DELETED ? fileDiff.oldPath : fileDiff.newPath
  const language = getLanguage(path)

  const stats = useMemo(() => {
    let added = 0
    let deleted = 0
    for (const hunk of fileDiff.hunks) {
      added += hunk.stats?.added || 0
      deleted += hunk.stats?.deleted || 0
    }
    return { added, deleted }
  }, [fileDiff.hunks])

  if (fileDiff.isBinary) {
    return (
      <div className="diff-view">
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

  return (
    <div className="diff-view">
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
          <div className="diff-view-toggle">
            <button
              className={`diff-view-button ${viewMode === 'unified' ? 'active' : ''}`}
              onClick={() => setViewMode('unified')}
            >
              Unified
            </button>
            <button
              className={`diff-view-button ${viewMode === 'split' ? 'active' : ''}`}
              onClick={() => setViewMode('split')}
            >
              Split
            </button>
          </div>
        </div>
      </div>

      <div className="diff-content">
        {fileDiff.hunks.length === 0 ? (
          <div className="diff-empty-notice">No changes in this file</div>
        ) : viewMode === 'unified' ? (
          <table className="diff-table diff-table-unified">
            <tbody>
              {fileDiff.hunks.map((hunk, hunkIdx) => (
                <Fragment key={hunkIdx}>
                  <tr className="diff-hunk-header-row">
                    <td colSpan={4} className="diff-hunk-header">
                      <HunkHeader header={hunk.header} />
                    </td>
                  </tr>
                  {hunk.lines.map((line, lineIdx) => (
                    <UnifiedLine
                      key={`${hunkIdx}-${lineIdx}`}
                      line={line}
                      language={language}
                    />
                  ))}
                </Fragment>
              ))}
            </tbody>
          </table>
        ) : (
          <SplitView hunks={fileDiff.hunks} language={language} />
        )}
      </div>
    </div>
  )
}

export default DiffView
