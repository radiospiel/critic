interface HelpModalProps {
  onClose: () => void
}

function HelpModal({ onClose }: HelpModalProps) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Keyboard Shortcuts</h2>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>
        <div className="modal-body">
          <section className="shortcut-section">
            <h3>Navigation</h3>
            <div className="shortcut-list">
              <div className="shortcut-item">
                <kbd>Tab</kbd>
                <span>Switch focus between file list and diff view</span>
              </div>
              <div className="shortcut-item">
                <kbd>↑</kbd> / <kbd>k</kbd>
                <span>Move selection up</span>
              </div>
              <div className="shortcut-item">
                <kbd>↓</kbd> / <kbd>j</kbd>
                <span>Move selection down</span>
              </div>
              <div className="shortcut-item">
                <kbd>Alt</kbd> + <kbd>↑</kbd>/<kbd>↓</kbd>
                <span>Jump 10 lines up/down</span>
              </div>
              <div className="shortcut-item">
                <kbd>g</kbd>
                <span>Go to top</span>
              </div>
              <div className="shortcut-item">
                <kbd>G</kbd>
                <span>Go to bottom</span>
              </div>
            </div>
          </section>

          <section className="shortcut-section">
            <h3>Selection (Diff View)</h3>
            <div className="shortcut-list">
              <div className="shortcut-item">
                <kbd>Shift</kbd> + <kbd>↑</kbd>
                <span>Expand selection upward</span>
              </div>
              <div className="shortcut-item">
                <kbd>Shift</kbd> + <kbd>↓</kbd>
                <span>Expand selection downward</span>
              </div>
            </div>
          </section>

          <section className="shortcut-section">
            <h3>Comments</h3>
            <div className="shortcut-list">
              <div className="shortcut-item">
                <kbd>Enter</kbd>
                <span>Open comment editor on selected line</span>
              </div>
              <div className="shortcut-item">
                <kbd>Ctrl</kbd> + <kbd>S</kbd>
                <span>Save comment</span>
              </div>
              <div className="shortcut-item">
                <kbd>Esc</kbd>
                <span>Close comment editor</span>
              </div>
            </div>
          </section>

          <section className="shortcut-section">
            <h3>General</h3>
            <div className="shortcut-list">
              <div className="shortcut-item">
                <kbd>?</kbd>
                <span>Show/hide this help</span>
              </div>
              <div className="shortcut-item">
                <kbd>Esc</kbd>
                <span>Close modal</span>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}

export default HelpModal
