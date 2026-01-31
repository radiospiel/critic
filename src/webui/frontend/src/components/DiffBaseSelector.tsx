import { useState, useEffect, useCallback } from 'react'
import { getDiffBases, setDiffBase } from '../api/client'

interface DiffBaseSelectorProps {
  onBaseChange?: () => void
}

function DiffBaseSelector({ onBaseChange }: DiffBaseSelectorProps) {
  const [bases, setBases] = useState<string[]>([])
  const [currentBase, setCurrentBase] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [isOpen, setIsOpen] = useState(false)

  const fetchBases = useCallback(async () => {
    const result = await getDiffBases()
    if (!result.error) {
      setBases(result.bases)
      setCurrentBase(result.currentBase)
    }
  }, [])

  useEffect(() => {
    fetchBases()
  }, [fetchBases])

  const handleBaseSelect = async (base: string) => {
    if (base === currentBase) {
      setIsOpen(false)
      return
    }

    setLoading(true)
    const result = await setDiffBase(base)
    if (result.success) {
      setCurrentBase(base)
      onBaseChange?.()
    }
    setLoading(false)
    setIsOpen(false)
  }

  // Handle keyboard shortcut 'b' to toggle dropdown
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return
      }
      if ((e.target as HTMLElement)?.closest?.('.tiptap')) {
        return
      }

      if (e.key === 'b' || e.key === 'B') {
        e.preventDefault()
        setIsOpen((prev) => !prev)
      } else if (e.key === 'Escape' && isOpen) {
        e.preventDefault()
        setIsOpen(false)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen])

  if (bases.length === 0) {
    return null
  }

  // Truncate long base names for display
  const displayBase = (base: string) => {
    if (base.length > 12) {
      return base.slice(0, 10) + '...'
    }
    return base
  }

  return (
    <div className="diff-base-selector">
      <button
        className="diff-base-button"
        onClick={() => setIsOpen(!isOpen)}
        disabled={loading}
        title={"Current base: " + currentBase + " (press 'b' to change)"}
      >
        <span className="diff-base-label">Base:</span>
        <span className="diff-base-value">{displayBase(currentBase)}</span>
        <span className="diff-base-arrow">{isOpen ? '▲' : '▼'}</span>
      </button>
      {isOpen && (
        <div className="diff-base-dropdown">
          {bases.map((base) => (
            <button
              key={base}
              className={"diff-base-option" + (base === currentBase ? ' active' : '')}
              onClick={() => handleBaseSelect(base)}
              disabled={loading}
            >
              {base}
              {base === currentBase && <span className="check-mark">✓</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

export default DiffBaseSelector
