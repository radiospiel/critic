import { useState, useCallback, useEffect, useRef } from 'react'
import { ThemeProvider, useTheme } from './context/ThemeContext'
import FileList, { FilterType } from './components/FileList'
import DiffView from './components/DiffView'
import DiffBaseSelector from './components/DiffBaseSelector'
import HelpModal from './components/HelpModal'
import { criticClient, getConfig, ServerConfig } from './api/client'
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
  const [showHelp, setShowHelp] = useState(false)
  const [contextLines, setContextLines] = useState(3)
  const [currentLineNo, setCurrentLineNo] = useState<{ lineNoNew: number; lineNoOld: number } | null>(null)
  const [restoreLineNo, setRestoreLineNo] = useState<{ lineNoNew: number; lineNoOld: number } | null>(null)
  const [secondsSinceLoad, setSecondsSinceLoad] = useState(0)
  const [fileListFilter, setFileListFilter] = useState<FilterType>('files')
  const [serverConfig, setServerConfig] = useState<ServerConfig | null>(null)
  const loadTimeRef = useRef(Date.now())
  const { theme, toggleTheme } = useTheme()

  // Load server config on mount
  useEffect(() => {
    getConfig().then((result) => {
      if (result.config) {
        setServerConfig(result.config)
      }
    })
  }, [])

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

  useEffect(() => {
    loadFileList()
  }, [loadFileList])

  const loadFileDiff = useCallback((file: string, ctxLines?: number, preserveSelection?: boolean) => {
    setSelectedFile(file)
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
    criticClient
      .getDiff({ path: file, contextLines: lines })
      .then((response) => {
        setSelectedFileDiff(response.file || null)
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
    if (message.type === 'reload') {
      console.log('Reload triggered by git change')
      // Reset the timer
      loadTimeRef.current = Date.now()
      setSecondsSinceLoad(0)
      // Reload the file list
      loadFileList()
      // Reload the current file diff if one is selected
      if (selectedFile) {
        loadFileDiff(selectedFile, contextLines, true)
      }
    } else if (message.type === 'file-changed' && message.path) {
      console.log('File changed:', message.path)
      // Reset the timer
      loadTimeRef.current = Date.now()
      setSecondsSinceLoad(0)
      // Reload the diff if it's the currently selected file
      if (selectedFile === message.path) {
        loadFileDiff(selectedFile, contextLines, true)
      }
    }
  }, [loadFileList, loadFileDiff, selectedFile, contextLines])

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
    loadFileDiff(file)
  }, [loadFileDiff])

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

      if (e.key === 'Tab') {
        e.preventDefault()
        setFocusedPanel((prev) => (prev === 'fileList' ? 'diffView' : 'fileList'))
      } else if (e.key === '?' || (e.shiftKey && e.key === '/')) {
        e.preventDefault()
        setShowHelp((prev) => !prev)
      } else if (e.key === 'Escape' && showHelp) {
        e.preventDefault()
        setShowHelp(false)
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
  }, [showHelp, handleIncreaseContext, handleDecreaseContext])

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="app-header">
          <span>Critic</span>
          <span className="render-timestamp">{secondsSinceLoad}s</span>
          <div className="header-buttons">
            <DiffBaseSelector onBaseChange={handleBaseChange} />
            <button className="help-button" onClick={() => setShowHelp(true)} title="Keyboard shortcuts">
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
        <FileList
          files={files}
          selectedFile={selectedFile}
          onSelectFile={handleSelectFile}
          isFocused={focusedPanel === 'fileList'}
          onFocus={() => setFocusedPanel('fileList')}
          onFilterChange={setFileListFilter}
        />
      </aside>
      <main className="main-content">
        {loading ? (
          <div className="empty-state">
            <span>Loading...</span>
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
            serverConfig={serverConfig}
          />
        ) : (
          <div className="empty-state">
            <span>
              {fileListFilter === 'conversations'
                ? 'To start a conversation in a changed source file, press Return on a source line and start writing your review message.'
                : fileListFilter === 'hidden'
                  ? 'Files can be hidden per .criticignore'
                  : 'Select a file to view changes'}
            </span>
          </div>
        )}
      </main>
      {showHelp && <HelpModal onClose={() => setShowHelp(false)} />}
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
