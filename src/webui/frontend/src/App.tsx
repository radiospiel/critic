import { useState } from 'react'
import { ThemeProvider, useTheme } from './context/ThemeContext'
import FileList from './components/FileList'
import DiffView from './components/DiffView'
import { criticClient } from './api/client'
import { FileDiff, FileSummary } from './gen/critic_pb'

function AppContent() {
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [selectedFileDiff, setSelectedFileDiff] = useState<FileDiff | null>(null)
  const [loading, setLoading] = useState(false)
  const { theme, toggleTheme } = useTheme()

  const handleSelectFile = (file: string, _fileSummary: FileSummary) => {
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
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="app-header">
          <span>Critic</span>
          <button className="theme-toggle" onClick={toggleTheme}>
            {theme === 'light' ? 'Dark' : 'Light'}
          </button>
        </div>
        <FileList selectedFile={selectedFile} onSelectFile={handleSelectFile} />
      </aside>
      <main className="main-content">
        {loading ? (
          <div className="empty-state">
            <span>Loading...</span>
          </div>
        ) : selectedFileDiff ? (
          <DiffView fileDiff={selectedFileDiff} />
        ) : (
          <div className="empty-state">
            <span>Select a file to view changes</span>
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
