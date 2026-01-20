package tui

import (
	"math/rand"

	"git.15b.it/eno/critic/teapot"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MatrixScreensaver displays a Matrix-style rain effect with kanji characters.
type MatrixScreensaver struct {
	teapot.BaseView

	width    int
	height   int
	columns  []*matrixColumn
	active   bool
	onDone   func() // Callback when screensaver is dismissed
	rng      *rand.Rand
	lastTick int64
}

// matrixColumn represents a single column of falling characters
type matrixColumn struct {
	chars     []rune  // Characters in the column (from top to bottom)
	head      int     // Position of the "head" (leading character)
	length    int     // Length of the visible trail
	speed     int     // How many ticks between updates
	tickCount int     // Counter for speed
	active    bool    // Whether this column is currently running
	startDelay int    // Ticks to wait before starting
}

// Katakana and other Matrix-style characters
var matrixChars = []rune{
	// Katakana
	'ア', 'イ', 'ウ', 'エ', 'オ',
	'カ', 'キ', 'ク', 'ケ', 'コ',
	'サ', 'シ', 'ス', 'セ', 'ソ',
	'タ', 'チ', 'ツ', 'テ', 'ト',
	'ナ', 'ニ', 'ヌ', 'ネ', 'ノ',
	'ハ', 'ヒ', 'フ', 'ヘ', 'ホ',
	'マ', 'ミ', 'ム', 'メ', 'モ',
	'ヤ', 'ユ', 'ヨ',
	'ラ', 'リ', 'ル', 'レ', 'ロ',
	'ワ', 'ヲ', 'ン',
	// Half-width katakana
	'ｱ', 'ｲ', 'ｳ', 'ｴ', 'ｵ',
	'ｶ', 'ｷ', 'ｸ', 'ｹ', 'ｺ',
	// Numbers
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	// Some ASCII symbols
	'@', '#', '$', '%', '&', '*', '+', '-', '=',
}

// NewMatrixScreensaver creates a new Matrix screensaver
func NewMatrixScreensaver() *MatrixScreensaver {
	m := &MatrixScreensaver{
		BaseView: teapot.NewBaseView(),
		rng:      rand.New(rand.NewSource(rand.Int63())),
	}
	m.SetFocusable(true)
	return m
}

// SetOnDone sets the callback to call when the screensaver is dismissed
func (m *MatrixScreensaver) SetOnDone(onDone func()) {
	m.onDone = onDone
}

// Start activates the screensaver
func (m *MatrixScreensaver) Start(width, height int) {
	m.width = width
	m.height = height
	m.active = true
	m.lastTick = teapot.GlobalTickCount

	// Initialize columns
	// Use every other column since katakana characters are often double-width
	numColumns := width / 2
	if numColumns < 1 {
		numColumns = 1
	}

	m.columns = make([]*matrixColumn, numColumns)
	for i := 0; i < numColumns; i++ {
		m.columns[i] = m.newColumn()
		// Stagger the start times for a more natural effect
		m.columns[i].startDelay = m.rng.Intn(height)
	}

	// Subscribe to global ticks for animation
	teapot.SubscribeToGlobalTicks(m)
}

// Stop deactivates the screensaver
func (m *MatrixScreensaver) Stop() {
	m.active = false
	teapot.UnsubscribeFromGlobalTicks(m)
}

// IsActive returns whether the screensaver is currently running
func (m *MatrixScreensaver) IsActive() bool {
	return m.active
}

// newColumn creates a new matrix column with random properties
func (m *MatrixScreensaver) newColumn() *matrixColumn {
	length := m.height/3 + m.rng.Intn(m.height/2+1)
	if length < 3 {
		length = 3
	}

	chars := make([]rune, m.height+length)
	for i := range chars {
		chars[i] = matrixChars[m.rng.Intn(len(matrixChars))]
	}

	return &matrixColumn{
		chars:     chars,
		head:      -length, // Start above the screen
		length:    length,
		speed:     1 + m.rng.Intn(2), // 1-2 ticks per move
		tickCount: 0,
		active:    true,
	}
}

// resetColumn resets a column that has gone off-screen
func (m *MatrixScreensaver) resetColumn(col *matrixColumn) {
	col.head = -col.length
	col.speed = 1 + m.rng.Intn(2)
	col.startDelay = m.rng.Intn(m.height / 2)

	// Regenerate characters
	for i := range col.chars {
		col.chars[i] = matrixChars[m.rng.Intn(len(matrixChars))]
	}
}

// HandleTick implements teapot.TickHandler
func (m *MatrixScreensaver) HandleTick() {
	if !m.active {
		return
	}

	// Update columns
	for _, col := range m.columns {
		if col.startDelay > 0 {
			col.startDelay--
			continue
		}

		col.tickCount++
		if col.tickCount >= col.speed {
			col.tickCount = 0
			col.head++

			// Occasionally change a random character in the trail
			if m.rng.Intn(10) == 0 {
				idx := m.rng.Intn(len(col.chars))
				col.chars[idx] = matrixChars[m.rng.Intn(len(matrixChars))]
			}

			// Reset column if it's completely off-screen
			if col.head-col.length > m.height {
				m.resetColumn(col)
			}
		}
	}

	m.Repaint()
}

// MightBeDirty always returns true since we're animating
func (m *MatrixScreensaver) MightBeDirty() bool {
	return m.active
}

// HandleKey handles keyboard input - any key dismisses the screensaver
func (m *MatrixScreensaver) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	if m.active {
		m.Stop()
		if m.onDone != nil {
			m.onDone()
		}
		return true, nil
	}
	return false, nil
}

// Render renders the Matrix rain effect
func (m *MatrixScreensaver) Render(buf *teapot.SubBuffer) {
	if !m.active {
		return
	}

	// Fill with black background
	blackCell := teapot.Cell{
		Rune:  ' ',
		Style: lipgloss.NewStyle().Background(lipgloss.Color("0")),
	}
	buf.Fill(buf.Bounds(), blackCell)

	// Render each column
	for colIdx, col := range m.columns {
		if col.startDelay > 0 {
			continue
		}

		// Calculate x position (double-width for katakana)
		x := colIdx * 2

		for y := 0; y < m.height; y++ {
			distFromHead := col.head - y

			// Skip if this position is not in the visible trail
			if distFromHead < 0 || distFromHead >= col.length {
				continue
			}

			// Get the character for this position
			charIdx := y % len(col.chars)
			char := col.chars[charIdx]

			// Calculate color based on distance from head
			style := m.getCharStyle(distFromHead, col.length)

			// Write the character
			buf.SetString(x, y, string(char), style)
		}
	}
}

// getCharStyle returns the style for a character based on its position in the trail
func (m *MatrixScreensaver) getCharStyle(distFromHead, length int) lipgloss.Style {
	// Head character is bright white/green
	if distFromHead == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")). // Bright white
			Bold(true).
			Background(lipgloss.Color("0"))
	}

	// Calculate brightness based on distance from head
	// Closer to head = brighter green
	ratio := float64(distFromHead) / float64(length)

	var colorCode string
	if ratio < 0.2 {
		colorCode = "46" // Bright green
	} else if ratio < 0.4 {
		colorCode = "40" // Green
	} else if ratio < 0.6 {
		colorCode = "34" // Dark green
	} else if ratio < 0.8 {
		colorCode = "28" // Darker green
	} else {
		colorCode = "22" // Very dark green
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorCode)).
		Background(lipgloss.Color("0"))
}
