import { useState, useEffect, useCallback } from 'react'
import { getDiffBases, setDiffRange } from '../api/client'

interface BranchSelection {
  start: string
  end: string
  soloMode: string | null
}

interface DiffBaseSelectorProps {
  onBaseChange?: () => void
}

function DiffBaseSelector({ onBaseChange }: DiffBaseSelectorProps) {
  const [bases, setBases] = useState<string[]>([])
  const [branchSelection, setBranchSelection] = useState<BranchSelection>({ start: '', end: '', soloMode: null })
  const [loading, setLoading] = useState(false)
  const [panelOpen, setPanelOpen] = useState(false)

  const fetchBases = useCallback(async () => {
    const result = await getDiffBases()
    if (!result.error) {
      setBases(result.bases)
      setBranchSelection({ start: result.currentStart, end: result.currentEnd, soloMode: null })
    }
  }, [])

  useEffect(() => {
    fetchBases()
  }, [fetchBases])

  const { start: currentStart, end: currentEnd, soloMode } = branchSelection

  const applyRange = async (newStart: string, newEnd: string, newSoloRef: string | null) => {
    if (newStart === currentStart && newEnd === currentEnd) {
      setBranchSelection({ start: newStart, end: newEnd, soloMode: newSoloRef })
      return
    }
    setLoading(true)
    const result = await setDiffRange(newStart, newEnd)
    if (result.success) {
      setBranchSelection({ start: newStart, end: newEnd, soloMode: newSoloRef })
      onBaseChange?.()
    }
    setLoading(false)
  }

  // Logical order: bases (oldest first) then '' (working dir).
  // This order is used for range semantics (start < end in index terms).
  const fullList = [...bases, '']

  // Escape closes the panel
  useEffect(() => {
    if (!panelOpen) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        setPanelOpen(false)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [panelOpen])

  if (bases.length === 0) return null

  const displayBase = (base: string) => {
    if (!base) return '(working dir)'
    return base
  }

  const startIdx = fullList.indexOf(currentStart)
  const endIdx = fullList.indexOf(currentEnd)
  const isSoloMode = soloMode !== null

  // Display list: reversed so newest (working dir) is at top, oldest at bottom.
  const displayList = [...fullList].reverse()

  return (
    <div className="diff-base-selector">
      <button
        className={'diff-base-trigger' + (panelOpen ? ' open' : '')}
        onClick={() => setPanelOpen(!panelOpen)}
        disabled={loading}
        title="Branch range"
      >
        <span className="diff-base-value">{displayBase(currentStart)}</span>
        <span className="diff-range-separator">&rarr;</span>
        <span className="diff-base-value">{displayBase(currentEnd)}</span>
        <span className="diff-base-chevron">{panelOpen ? '\u25B4' : '\u25BE'}</span>
      </button>

      {panelOpen && (
        <div className="diff-graph-panel">
          <div className="diff-graph-hint">Click to toggle between changes in a single branch or all changes</div>
          {displayList.filter(v => v !== '').map((value, displayIdx) => {
            const logicalIdx = fullList.indexOf(value)
            const isStart = value === currentStart
            const isEnd = value === currentEnd
            const inRange = logicalIdx >= startIdx && logicalIdx <= endIdx && (!isSoloMode || isStart)
            const isLast = displayIdx === displayList.length - 1
            const label = value || '(working dir)'

            const hasNewer = logicalIdx + 1 < fullList.length
            const soloEnd = hasNewer ? fullList[logicalIdx + 1] : ''
            const isSolo = value === soloMode
            const clickable = value !== '' && hasNewer

            const handleRowClick = () => {
              if (loading || !clickable) return
              if (value === currentStart) {
                // Toggle mode on current branch
                if (isSoloMode) {
                  applyRange(value, '', null)
                } else {
                  applyRange(value, soloEnd, value)
                }
              } else {
                // Activate this branch in current mode
                if (isSoloMode) {
                  applyRange(value, soloEnd, value)
                } else {
                  applyRange(value, '', null)
                }
              }
            }

            return (
              <div
                key={value || '__workdir__'}
                className={`diff-graph-row${isStart ? ' selected-start' : ''}${isEnd ? ' selected-end' : ''}${inRange ? ' in-range' : ''}${isSolo ? ' solo' : ''}${isLast ? ' last' : ''}${!clickable ? ' inert' : ''}`}
                onClick={handleRowClick}
              >
                <span className="diff-graph-gutter">
                  {inRange && !isSoloMode && <span className={`diff-graph-line${isStart ? ' start' : ''}${isEnd ? ' end' : ''}`} />}
                  {isSoloMode && isSolo ? (
                    <span className="diff-graph-eye"><svg width="14" height="10" viewBox="0 0 14 10" fill="none"><path d="M7 1C3.5 1 1 5 1 5s2.5 4 6 4 6-4 6-4-2.5-4-6-4Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round"/><circle cx="7" cy="5" r="1.8" fill="currentColor"/></svg></span>
                  ) : (
                    <>
                      {isStart && !isSoloMode && <span className="diff-graph-arrow up" />}
                      {isEnd && !isSoloMode && <span className="diff-graph-arrow down" />}
                    </>
                  )}
                </span>
                <span className="diff-graph-label" title={label}>{label}</span>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

export default DiffBaseSelector
