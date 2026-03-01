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

  // Full ordered list: bases (oldest first) then '' (working dir)
  // bases come from the API in order, with the oldest/most-ancestral first.
  const fullList = [...bases, '']

  const handleNodeClick = (value: string) => {
    if (loading) return
    const clickIdx = fullList.indexOf(value)
    const startIdx = fullList.indexOf(currentStart)
    const endIdx = fullList.indexOf(currentEnd)
    if (clickIdx < 0 || startIdx < 0 || endIdx < 0) return

    if (clickIdx === startIdx) {
      // Click on start: shrink range (move start inward), unless neighbors
      if (startIdx + 1 >= endIdx) return
      applyRange(fullList[startIdx + 1], currentEnd)
    } else if (clickIdx === endIdx) {
      // Click on end: shrink range (move end inward), unless neighbors
      if (endIdx - 1 <= startIdx) return
      applyRange(currentStart, fullList[endIdx - 1])
    } else if (clickIdx > startIdx && clickIdx < endIdx) {
      // Inside range: set start to clicked branch
      applyRange(value, currentEnd)
    } else if (clickIdx < startIdx) {
      // Before range: expand start outward
      applyRange(value, currentEnd)
    } else if (clickIdx > endIdx) {
      // After range: expand end outward
      applyRange(currentStart, value)
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
    if (!base) return 'working dir'
    return base
  }

  const startIdx = fullList.indexOf(currentStart)
  const endIdx = fullList.indexOf(currentEnd)

  return (
    <div className="diff-base-selector">
      <button
        className={'diff-base-trigger' + (panelOpen ? ' open' : '')}
        onClick={() => setPanelOpen(!panelOpen)}
        disabled={loading}
        title="Branch range"
      >
        <span className="diff-base-value">{displayBase(currentStart)}</span>
        <span className="diff-range-separator">→</span>
        <span className="diff-base-value">{displayBase(currentEnd)}</span>
      </button>

      {panelOpen && (
        <div className="diff-graph-panel">
          <div className="diff-graph-hint">Click on branch names to adjust selection</div>
          {fullList.map((value, idx) => {
            const isStart = value === currentStart
            const isEnd = value === currentEnd
            const inRange = idx >= startIdx && idx <= endIdx
            const isLast = idx === fullList.length - 1
            const label = value || 'working dir'

            return (
              <div
                key={value || '__workdir__'}
                className={`diff-graph-row${isStart ? ' selected-start' : ''}${isEnd ? ' selected-end' : ''}${inRange ? ' in-range' : ''}${isLast ? ' last' : ''}`}
                onClick={() => handleNodeClick(value)}
              >
                <span className="diff-graph-gutter">
                  {inRange && <span className={`diff-graph-line${isStart ? ' start' : ''}${isEnd ? ' end' : ''}`} />}
                  {isStart && <span className="diff-graph-arrow down" />}
                  {isEnd && <span className="diff-graph-arrow up" />}
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
