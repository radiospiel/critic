import { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { ThemeProvider, useTheme } from './context/ThemeContext'
import FileList, { FilterType } from './components/FileList'
import DiffView from './components/DiffView'
import DiffBaseSelector from './components/DiffBaseSelector'
import CommentDisplay from './components/CommentDisplay'
import { criticClient, getConfig, getConversations, getRootConversation, ServerConfig, CommentConversation } from './api/client'
import { FileDiff, FileSummary, FileStatus } from './gen/critic_pb'
import { useWebSocket } from './hooks/useWebSocket'

type FocusedPanel = 'fileList' | 'diffView'

function getFilePath(file: FileSummary): string {
  return file.status === FileStatus.DELETED ? file.oldPath : file.newPath
}

function AppContent() {
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [selectedFileDiff, setSelectedFileDiff] = useState<FileDiff | null>(null)
  const [loading, setLoading] = useState(false)
  const [files, setFiles] = useState<FileSummary[]>([])
  const [focusedPanel, setFocusedPanel] = useState<FocusedPanel>('fileList')
  const [contextLines, setContextLines] = useState(3)
  const [currentLineNo, setCurrentLineNo] = useState<{ lineNoNew: number; lineNoOld: number } | null>(null)
  const [restoreLineNo, setRestoreLineNo] = useState<{ lineNoNew: number; lineNoOld: number } | null>(null)
  const [secondsSinceLoad, setSecondsSinceLoad] = useState(0)
  const [fileListFilter, setFileListFilter] = useState<FilterType>('automatic')
  const [serverConfig, setServerConfig] = useState<ServerConfig | null>(null)
  const [rootConversation, setRootConversation] = useState<CommentConversation | null>(null)
  const [showRootConversation, setShowRootConversation] = useState(false)
  const [showArchived, setShowArchived] = useState(false)
  const [allConversations, setAllConversations] = useState<CommentConversation[]>([])
  const loadTimeRef = useRef(Date.now())
  const { theme, toggleTheme } = useTheme()

  // Load server config and project config on mount
  useEffect(() => {
    getConfig().then((result) => {
      if (result.config) {
        setServerConfig(result.config)
      }
    })
  }, [])

  // Load root conversation (announcements)
  const loadRootConversation = useCallback(() => {
    getRootConversation().then((result) => {
      setRootConversation(result.conversation)
    })
  }, [])

  useEffect(() => {
    loadRootConversation()
  }, [loadRootConversation])

  // Timer to update seconds since last reload
  useEffect(() => {
    const interval = setInterval(() => {
      setSecondsSinceLoad(Math.floor((Date.now() - loadTimeRef.current) / 1000))
    }, 1000)
    return () => clearInterval(interval)
  }, [])

  // Load file list for navigation
  const loadFileList = useCallback(() => {
    criticClient
      .getDiffSummary({})
      .then((response) => {
        setFiles(response.diff?.files || [])
      })
      .catch((err) => {
        console.error('Failed to load file list:', err)
      })
  }, [])

  // Load all conversations in a single request
  const loadAllConversations = useCallback(() => {
    getConversations().then((result) => {
      setAllConversations(result.conversations)
    })
  }, [])

  useEffect(() => {
    loadFileList()
    loadAllConversations()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const loadFileDiff = useCallback((file: string, ctxLines?: number, preserveSelection?: boolean) => {
    setSelectedFile(file)
    setShowRootConversation(false)
    // Only show loading indicator when changing files, not when changing context
    if (!preserveSelection) {
      setLoading(true)
      setRestoreLineNo(null)
      // Tell server to watch this file for changes
      criticClient.watchFile({ path: file }).catch((err) => {
        console.error('Failed to set watch file:', err)
      })
    }
    const lines = ctxLines ?? contextLines
    criticClient.getDiff({ path: file, contextLines: lines })
      .then((diffResponse) => {
        setSelectedFileDiff(diffResponse.file || null)
        setLoading(false)
      })
      .catch((err) => {
        console.error('Failed to load diff:', err)
        setSelectedFileDiff(null)
        setLoading(false)
      })
  }, [contextLines])

  // Handle WebSocket messages for live reload
  const handleWebSocketMessage = useCallback((message: { type: string; path?: string }) => {
    // Don't reload while user is typing in the TipTap editor
    if (document.activeElement?.closest('.tiptap')) {
      return
    }

    if (message.type === 'reload') {
      console.log('Reload triggered by git change')
      // Reset the timer
      loadTimeRef.current = Date.now()
      setSecondsSinceLoad(0)
      // Reload the file list, conversations, and root conversation
      loadFileList()
      loadAllConversations()
      loadRootConversation()
      // Reload the current file diff if one is selected
      if (selectedFile) {
        loadFileDiff(selectedFile, contextLines, true)
      }
    } else if (message.type === 'file-changed' && message.path) {
      console.log('File changed:', message.path)
      // Reset the timer
      loadTimeRef.current = Date.now()
      setSecondsSinceLoad(0)
      // Reload conversations and the diff if it's the currently selected file
      loadAllConversations()
      if (selectedFile === message.path) {
        loadFileDiff(selectedFile, contextLines, true)
      }
    }
  }, [loadFileList, loadFileDiff, loadAllConversations, loadRootConversation, selectedFile, contextLines])

  useWebSocket(handleWebSocketMessage)

  // Handle base change - reload file list and preserve current file if still present
  const handleBaseChange = useCallback(() => {
    const previousFile = selectedFile
    setSelectedFileDiff(null)
    // Wait a moment for the backend to update the diff
    setTimeout(() => {
      criticClient
        .getDiffSummary({})
        .then((response) => {
          const newFiles = response.diff?.files || []
          setFiles(newFiles)

          if (newFiles.length === 0) {
            setSelectedFile(null)
            return
          }

          // Check if the previously selected file still exists
          const previousFileExists = previousFile && newFiles.some((f) => getFilePath(f) === previousFile)

          if (previousFileExists) {
            // Reload the same file with new diff base
            loadFileDiff(previousFile)
          } else {
            // Select the first file
            const firstFile = getFilePath(newFiles[0])
            loadFileDiff(firstFile)
          }
        })
        .catch((err) => {
          console.error('Failed to load file list:', err)
        })
    }, 100)
  }, [selectedFile, loadFileDiff])

  const handleSelectFile = useCallback((file: string, _fileSummary: FileSummary) => {
    if (file === selectedFile) {
      setShowRootConversation(false)
      return
    }
    loadFileDiff(file)
  }, [loadFileDiff, selectedFile])

  const handleScrollToLine = useCallback((lineNo: number) => {
    setRestoreLineNo({ lineNoNew: lineNo, lineNoOld: 0 })
  }, [])

  const handleSelectRootConversation = useCallback(() => {
    setSelectedFile(null)
    setSelectedFileDiff(null)
    setShowRootConversation(true)
  }, [])

  const handleNavigatePrevFile = useCallback(() => {
    if (files.length === 0 || !selectedFile) return
    const currentIndex = files.findIndex((f) => getFilePath(f) === selectedFile)
    if (currentIndex > 0) {
      const prevFile = files[currentIndex - 1]
      loadFileDiff(getFilePath(prevFile))
    }
  }, [files, selectedFile, loadFileDiff])

  const handleNavigateNextFile = useCallback(() => {
    if (files.length === 0 || !selectedFile) return
    const currentIndex = files.findIndex((f) => getFilePath(f) === selectedFile)
    if (currentIndex < files.length - 1) {
      const nextFile = files[currentIndex + 1]
      loadFileDiff(getFilePath(nextFile))
    }
  }, [files, selectedFile, loadFileDiff])

  const handleIncreaseContext = useCallback(() => {
    if (!selectedFile) return
    const newLines = contextLines + 3
    setContextLines(newLines)
    setRestoreLineNo(currentLineNo)
    loadFileDiff(selectedFile, newLines, true)
  }, [selectedFile, contextLines, loadFileDiff, currentLineNo])

  const handleDecreaseContext = useCallback(() => {
    if (!selectedFile) return
    const newLines = Math.max(3, contextLines - 3)
    if (newLines !== contextLines) {
      setContextLines(newLines)
      setRestoreLineNo(currentLineNo)
      loadFileDiff(selectedFile, newLines, true)
    }
  }, [selectedFile, contextLines, loadFileDiff, currentLineNo])

  const handleResetContext = useCallback(() => {
    if (!selectedFile || contextLines === 3) return
    setContextLines(3)
    setRestoreLineNo(currentLineNo)
    loadFileDiff(selectedFile, 3, true)
  }, [selectedFile, contextLines, loadFileDiff, currentLineNo])

  const handleSelectionChange = useCallback((lineNoNew: number, lineNoOld: number) => {
    setCurrentLineNo({ lineNoNew, lineNoOld })
  }, [])

  // Filter conversations for the selected file (derived from single bulk load)
  const selectedFileConversations = useMemo(
    () => selectedFile
      ? allConversations.filter((c) => c.filePath === selectedFile)
      : [],
    [allConversations, selectedFile]
  )

  // Global keyboard handler for Tab, ?, and context line keys
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't handle if in input field
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return
      }
      // Don't handle if in tiptap editor
      if ((e.target as HTMLElement)?.closest?.('.tiptap')) {
        return
      }

      // Let browser handle Cmd+key shortcuts (copy, paste, select all, etc.)
      if (e.metaKey) {
        return
      }

      // Ctrl+1 = conversations, Ctrl+2 = files
      if (e.ctrlKey && e.key === '1') {
        e.preventDefault()
        setFileListFilter('conversations')
        return
      }
      if (e.ctrlKey && e.key === '2') {
        e.preventDefault()
        setFileListFilter('files')
        return
      }

      // Let browser handle other Ctrl+key shortcuts
      if (e.ctrlKey) {
        return
      }

      if (e.key === 'Tab') {
        e.preventDefault()
        setFocusedPanel((prev) => (prev === 'fileList' ? 'diffView' : 'fileList'))
      } else if (e.key === '?' || (e.shiftKey && e.key === '/')) {
        e.preventDefault()
        setSelectedFile(null)
        setSelectedFileDiff(null)
        setShowRootConversation(false)
      } else if (e.key === 'c') {
        e.preventDefault()
        handleIncreaseContext()
      } else if (e.key === 'C') {
        e.preventDefault()
        handleDecreaseContext()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleIncreaseContext, handleDecreaseContext])

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="app-header">
          <span>Critic</span>
          <span className="render-timestamp">{secondsSinceLoad}s</span>
          <div className="header-buttons">
            <button className="help-button" onClick={() => { setSelectedFile(null); setSelectedFileDiff(null); setShowRootConversation(false) }} title="Keyboard shortcuts">
              ?
            </button>
            <button className="theme-toggle" onClick={toggleTheme} title={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}>
              {theme === 'light' ? (
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
                </svg>
              ) : (
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="12" y1="1" x2="12" y2="3" />
                  <line x1="12" y1="21" x2="12" y2="23" />
                  <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                  <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                  <line x1="1" y1="12" x2="3" y2="12" />
                  <line x1="21" y1="12" x2="23" y2="12" />
                  <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
                  <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
                  <circle cx="12" cy="12" r="5" />
                </svg>
              )}
            </button>
          </div>
        </div>
        <DiffBaseSelector onBaseChange={handleBaseChange} />
        <FileList
          files={files}
          allConversations={allConversations}
          selectedFile={selectedFile}
          onSelectFile={handleSelectFile}
          onSelectRootConversation={handleSelectRootConversation}
          isFocused={focusedPanel === 'fileList'}
          onFocus={() => setFocusedPanel('fileList')}
          filter={fileListFilter}
          onFilterChange={setFileListFilter}
          showArchived={showArchived}
          onShowArchivedChange={setShowArchived}
          rootConversation={rootConversation}
          isRootConversationSelected={showRootConversation}
          onConversationsChanged={() => { loadAllConversations(); loadFileList(); loadRootConversation() }}
          onScrollToLine={handleScrollToLine}
        />
      </aside>
      <main className="main-content">
        {loading ? (
          <div className="empty-state">
            <span>Loading...</span>
          </div>
        ) : showRootConversation && rootConversation ? (
          <div className="root-conversation-view">
            <CommentDisplay
              conversations={[rootConversation]}
              lineNumber={rootConversation.lineNumber}
              onReplyAdded={loadRootConversation}
              scrollToBottom
              alwaysShowEditor
            />
          </div>
        ) : selectedFileDiff ? (
          <DiffView
            fileDiff={selectedFileDiff}
            onNavigatePrevFile={handleNavigatePrevFile}
            onNavigateNextFile={handleNavigateNextFile}
            isFocused={focusedPanel === 'diffView'}
            onFocus={() => setFocusedPanel('diffView')}
            contextLines={contextLines}
            onIncreaseContext={handleIncreaseContext}
            onDecreaseContext={handleDecreaseContext}
            onResetContext={handleResetContext}
            onSelectionChange={handleSelectionChange}
            restoreLineNo={restoreLineNo}
            showOnlyConversations={fileListFilter === 'conversations'}
            showArchived={showArchived}
            serverConfig={serverConfig}
            conversations={selectedFileConversations}
            onConversationsChanged={() => { loadAllConversations(); loadFileList() }}
          />
        ) : (
          <div className="empty-state usage-guide">
            <h2>Getting Started with Critic</h2>
            <section>
              <h3>Installing Critic</h3>
              <p>Critic is tested to work on macOS. It needs recent go and npm installations.</p>
              <pre><code>git clone radiospiel/critic{'\n'}make install</code></pre>
            </section>
            <section>
              <h3>Review Changes</h3>
              <p>Select a file from the sidebar to view its diff. Press <kbd>Enter</kbd> on any source line to start a conversation.</p>
            </section>
            <section>
              <h3>Connect Claude Code</h3>
              <p>Register the Critic MCP server so Claude can read and reply to your review comments:</p>
              <pre><code>claude mcp add critic -- critic mcp</code></pre>
              <p>Then use the slash commands:</p>
              <pre><code>/critic:summarize  — summarize changes and post via critic_announce{'\n'}/critic:step       — address unresolved feedback{'\n'}/critic:loop       — repeat step until all conversations are resolved</code></pre>
            </section>
            <section>
              <h3>VS Code Extension</h3>
              <p>Install the Critic extension to see review comments inline in VS Code:</p>
              <pre><code>curl -O {window.location.origin}/download/critic-vscode.vsix{'\n'}code --install-extension critic-vscode.vsix</code></pre>
            </section>
            <section>
              <h3>Keyboard Shortcuts</h3>
              <table className="shortcut-table">
                <tbody>
                  <tr><td colSpan={4} className="shortcut-table-heading">General</td></tr>
                  <tr><td><kbd>?</kbd></td><td>Show this help</td><td><kbd>g</kbd> / <kbd>G</kbd></td><td>Go to top / bottom</td></tr>
                  <tr><td><kbd>Tab</kbd></td><td>Switch file list / diff focus</td><td><kbd>Alt</kbd> + <kbd>↑</kbd>/<kbd>↓</kbd></td><td>Jump 25 lines</td></tr>
                  <tr><td><kbd>↑</kbd> / <kbd>k</kbd></td><td>Move selection up</td><td><kbd>Shift</kbd> + <kbd>↑</kbd>/<kbd>↓</kbd></td><td>Expand selection</td></tr>
                  <tr><td><kbd>↓</kbd> / <kbd>j</kbd></td><td>Move selection down</td><td></td><td></td></tr>

                  <tr><td colSpan={4} className="shortcut-table-heading">File List Sections</td></tr>
                  <tr>
                    <td><kbd>Ctrl</kbd> + <kbd>1</kbd></td><td>Conversations</td>
                    <td><kbd>Ctrl</kbd> + <kbd>2</kbd></td><td>Files</td>
                  </tr>

                  <tr><td colSpan={4} className="shortcut-table-heading">Diffs</td></tr>
                  <tr><td><kbd>Enter</kbd></td><td>Open comment editor</td><td><kbd>⌥</kbd> + <kbd>↵</kbd></td><td>Save comment</td></tr>
                  <tr><td><kbd>Esc</kbd></td><td>Close comment editor</td><td><kbd>c</kbd> / <kbd>C</kbd></td><td>Context lines +/-</td></tr>
                </tbody>
              </table>
            </section>
          </div>
        )}
      </main>
    </div>
  )
}

function App() {
  return (
    <ThemeProvider>
      <AppContent />
    </ThemeProvider>
  )
}

export default App
