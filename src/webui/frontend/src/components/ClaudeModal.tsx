import Markdown from 'react-markdown'

interface ClaudeModalProps {
  text: string
  onClose: () => void
}

export default function ClaudeModal({ text, onClose }: ClaudeModalProps) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-dialog" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span>Ask Claude</span>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>
        <div className="modal-body modal-markdown">
          <Markdown>{text}</Markdown>
        </div>
      </div>
    </div>
  )
}
