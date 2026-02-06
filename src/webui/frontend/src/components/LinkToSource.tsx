import { ServerConfig } from '../api/client'

// Check if we're on localhost
function isLocalhost(): boolean {
  return window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
}

// Link to open source file in IDE (only on localhost, when editor URL is configured)
interface LinkToSourceProps {
  lineNo: number
  filePath: string
  serverConfig?: ServerConfig | null
}

function LinkToSource({ lineNo, filePath, serverConfig }: LinkToSourceProps) {
  if (!isLocalhost() || !serverConfig?.editorUrl) {
    return <>{lineNo}</>
  }

  const href = serverConfig.editorUrl
    .replace('{file}', `${serverConfig.gitRoot}/${filePath}`)
    .replace('{line}', String(lineNo))

  return (
    <a
      href={href}
      onClick={(e) => e.stopPropagation()}
      title="Open in editor"
      className="line-number-link"
    >
      {lineNo}
    </a>
  )
}

export default LinkToSource
