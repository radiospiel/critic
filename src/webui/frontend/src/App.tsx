import { useState, useEffect } from 'react'
import FileList from './components/FileList'

function App() {
  const [selectedFile, setSelectedFile] = useState<string | null>(null)

  // Connect to WebSocket for live reload in dev mode
  useEffect(() => {
    if (import.meta.env.DEV) {
      const ws = new WebSocket(`ws://${window.location.host}/ws`)

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          if (msg.type === 'reload') {
            console.log('Backend changed, reloading...')
            window.location.reload()
          }
        } catch {
          // Ignore non-JSON messages (like pings)
        }
      }

      ws.onclose = () => {
        console.log('WebSocket closed, will retry...')
        // Retry connection after 2 seconds
        setTimeout(() => window.location.reload(), 2000)
      }

      return () => ws.close()
    }
  }, [])

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
