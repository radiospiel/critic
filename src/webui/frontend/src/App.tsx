import { useState, useCallback, useEffect } from 'react'
import { ThemeProvider, useTheme } from './context/ThemeContext'
import FileList from './components/FileList'
import DiffView from './components/DiffView'
import HelpModal from './components/HelpModal'
import { criticClient } from './api/client'
import { FileDiff, FileSummary, FileStatus } from './gen/critic_pb'

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
  const { theme, toggleTheme } = useTheme()

  // Load file list for navigation
  useEffect(() => {
    criticClient
      .getDiffSummary({})
      .then((response) => {
        setFiles(response.diff?.files || [])
      })
      .catch((err) => {
        console.error('Failed to load file list:', err)
      })
  }, [])

  const loadFileDiff = useCallback((file: string) => {
    setSelectedFile(file)
    setLoading(true)
    criticClient
      .getDiff({ path: file })
      .then((response) => {
        setSelectedFileDiff(response.file || null)
        setLoading(false)
      })
      .catch((err) => {
        console.error('Failed to load diff:', err)
        setSelectedFileDiff(null)
        setLoading(false)
      })
  }, [])

  const handleSelectFile = useCallback((file: string, _fileSummary: FileSummary) => {
    loadFileDiff(file)
    setFocusedPanel('diffView')
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

  // Global keyboard handler for Tab and ? keys
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't handle if in input field
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
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
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [showHelp])

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="app-header">
          <span>Critic</span>
          <div className="header-buttons">
            <button className="help-button" onClick={() => setShowHelp(true)} title="Keyboard shortcuts">
              ?
            </button>
            <button className="theme-toggle" onClick={toggleTheme}>
              {theme === 'light' ? 'Dark' : 'Light'}
            </button>
          </div>
        </div>
        <FileList
          selectedFile={selectedFile}
          onSelectFile={handleSelectFile}
          isFocused={focusedPanel === 'fileList'}
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
          />
        ) : (
          <div className="empty-state">
            <span>Select a file to view changes</span>
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
