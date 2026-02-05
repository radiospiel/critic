import { ServerConfig } from '../api/client'

// Check if we're on localhost
function isLocalhost(): boolean {
  return window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
}

// Link to open source file in IDE (only in dev mode on localhost)
interface LinkToSourceProps {
  lineNo: number
  filePath: string
  serverConfig?: ServerConfig | null
}

function LinkToSource({ lineNo, filePath, serverConfig }: LinkToSourceProps) {
  if (!isLocalhost()) {
    return <>{lineNo}</>
  }

  if (!serverConfig) {
    return <>{lineNo}</>
  }

  return (
    <a
      href={`goland://open?file=${serverConfig.gitRoot}/${filePath}&line=${lineNo}`}
      onClick={(e) => e.stopPropagation()}
      title="Open in GoLand"
      className="line-number-link"
    >
      {lineNo}
    </a>
  )
}

export default LinkToSource
