import { useState } from 'react'
import FileList from './components/FileList'

function App() {
  const [selectedFile, setSelectedFile] = useState<string | null>(null)

  return (
    <div className="app">
      <aside className="sidebar">
        <FileList
          selectedFile={selectedFile}
          onSelectFile={setSelectedFile}
        />
      </aside>
      <main className="main-content">
        {/* File content will be rendered here */}
      </main>
    </div>
  )
}

export default App
