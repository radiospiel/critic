import { useState, useEffect, useCallback } from 'react'
import { getDiffBases, setDiffRange } from '../api/client'

interface DiffBaseSelectorProps {
  onBaseChange?: () => void
}

function DiffBaseSelector({ onBaseChange }: DiffBaseSelectorProps) {
  const [bases, setBases] = useState<string[]>([])
  const [currentStart, setCurrentStart] = useState<string>('')
  const [currentEnd, setCurrentEnd] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [panelOpen, setPanelOpen] = useState(false)

  const fetchBases = useCallback(async () => {
    const result = await getDiffBases()
    if (!result.error) {
      setBases(result.bases)
      setCurrentStart(result.currentStart)
      setCurrentEnd(result.currentEnd)
    }
  }, [])

  useEffect(() => {
    fetchBases()
  }, [fetchBases])

  const applyRange = async (newStart: string, newEnd: string) => {
    if (newStart === currentStart && newEnd === currentEnd) return
    setLoading(true)
    const result = await setDiffRange(newStart, newEnd)
    if (result.success) {
      setCurrentStart(newStart)
      setCurrentEnd(newEnd)
      onBaseChange?.()
    }
    setLoading(false)
  }

  // Logical order: bases (oldest first) then '' (working dir).
  // This order is used for range semantics (start < end in index terms).
  const fullList = [...bases, '']

  const handleNodeClick = (value: string) => {
    if (loading) return
    const clickIdx = fullList.indexOf(value)
    const startIdx = fullList.indexOf(currentStart)
    const endIdx = fullList.indexOf(currentEnd)
    if (clickIdx < 0 || startIdx < 0 || endIdx < 0) return

    // Range click logic:
    //
    // The user selects a contiguous range [start..end] within fullList.
    // A minimum of 2 entries must always remain selected.
    //
    // - Clicking an INACTIVE entry (outside the range) expands the nearest
    //   edge to include it — and all entries in between.
    // - Clicking an ACTIVE entry (inside the range) shrinks the range:
    //     * Start edge  → removes it, moving start one step inward.
    //     * End edge    → removes it, moving end one step inward.
    //     * Middle      → removes it AND everything older (toward start),
    //                     keeping the newer side (toward end).
    const isInRange = clickIdx >= startIdx && clickIdx <= endIdx
    const rangeSize = endIdx - startIdx + 1

    if (isInRange) {
      // Clicking an active entry: deactivate it.
      // Guard: don't shrink below 2 active entries.
      if (rangeSize <= 2) return

      if (clickIdx === startIdx) {
        // Click on start: shrink range by moving start inward
        applyRange(fullList[startIdx + 1], currentEnd)
      } else if (clickIdx === endIdx) {
        // Click on end: shrink range by moving end inward
        applyRange(currentStart, fullList[endIdx - 1])
      } else {
        // Click in middle: deactivate clicked entry and everything toward
        // the start (older side), keeping the end (newer) side.
        applyRange(fullList[clickIdx + 1], currentEnd)
      }
    } else {
      // Clicking an inactive entry: activate it by expanding the range.
      if (clickIdx < startIdx) {
        applyRange(value, currentEnd)
      } else {
        applyRange(currentStart, value)
      }
    }
  }

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
      </button>

      {panelOpen && (
        <div className="diff-graph-panel">
          <div className="diff-graph-hint">Click on branch names to adjust selection</div>
          {displayList.map((value, displayIdx) => {
            const logicalIdx = fullList.indexOf(value)
            const isStart = value === currentStart
            const isEnd = value === currentEnd
            const inRange = logicalIdx >= startIdx && logicalIdx <= endIdx
            const isLast = displayIdx === displayList.length - 1
            const label = value || '(working dir)'

            return (
              <div
                key={value || '__workdir__'}
                className={`diff-graph-row${isStart ? ' selected-start' : ''}${isEnd ? ' selected-end' : ''}${inRange ? ' in-range' : ''}${isLast ? ' last' : ''}`}
                onClick={() => handleNodeClick(value)}
              >
                <span className="diff-graph-gutter">
                  {inRange && <span className={`diff-graph-line${isStart ? ' start' : ''}${isEnd ? ' end' : ''}`} />}
                  {isStart && <span className="diff-graph-arrow up" />}
                  {isEnd && <span className="diff-graph-arrow down" />}
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
